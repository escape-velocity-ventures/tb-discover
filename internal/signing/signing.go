// Package signing implements Ed25519 command signature verification
// for the tb-manage agent. The SaaS signs commands; the agent verifies.
package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MaxTimestampAge is the maximum age of a signed message before it's rejected.
const MaxTimestampAge = 30 * time.Second

// SignedEnvelope wraps any protocol message with signing fields.
type SignedEnvelope struct {
	Signature string `json:"signature,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Origin    string `json:"origin,omitempty"`
}

// SignedPayload is the canonical structure that gets signed.
type SignedPayload struct {
	Command   json.RawMessage `json:"command"`
	Timestamp int64           `json:"timestamp"`
	Nonce     string          `json:"nonce"`
	UserID    string          `json:"user_id"`
	Origin    string          `json:"origin"`
}

// Verifier checks Ed25519 signatures on incoming messages.
type Verifier struct {
	pubKey     ed25519.PublicKey
	nonceStore *NonceStore
}

// NewVerifier creates a Verifier with the given Ed25519 public key.
func NewVerifier(pubKey ed25519.PublicKey) *Verifier {
	return &Verifier{
		pubKey:     pubKey,
		nonceStore: NewNonceStore(MaxTimestampAge * 2),
	}
}

// ParsePublicKey decodes a hex or base64-encoded Ed25519 public key.
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty public key")
	}

	// Try hex first (64 hex chars = 32 bytes)
	if len(s) == 64 {
		b, err := hex.DecodeString(s)
		if err == nil && len(b) == ed25519.PublicKeySize {
			return ed25519.PublicKey(b), nil
		}
	}

	// Try base64
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		b, err := enc.DecodeString(s)
		if err == nil && len(b) == ed25519.PublicKeySize {
			return ed25519.PublicKey(b), nil
		}
	}

	return nil, fmt.Errorf("invalid public key: must be 32 bytes, hex or base64 encoded")
}

// VerificationResult contains the outcome of signature verification.
type VerificationResult struct {
	Valid     bool
	Reason    string
	UserID    string
	Origin    string
	Timestamp int64
}

// Verify checks the signature, timestamp, and nonce of a raw JSON message.
func (v *Verifier) Verify(raw []byte) (command []byte, result VerificationResult) {
	var env SignedEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, VerificationResult{Reason: fmt.Sprintf("invalid JSON: %v", err)}
	}

	if env.Signature == "" {
		return nil, VerificationResult{Reason: "missing signature"}
	}
	if env.Nonce == "" {
		return nil, VerificationResult{Reason: "missing nonce"}
	}

	result.UserID = env.UserID
	result.Origin = env.Origin
	result.Timestamp = env.Timestamp

	// Check timestamp freshness
	now := time.Now().Unix()
	age := now - env.Timestamp
	if age < 0 {
		age = -age
	}
	if age > int64(MaxTimestampAge.Seconds()) {
		return nil, VerificationResult{
			Reason:    fmt.Sprintf("timestamp too old or in future: age=%ds, max=%ds", age, int64(MaxTimestampAge.Seconds())),
			UserID:    env.UserID,
			Origin:    env.Origin,
			Timestamp: env.Timestamp,
		}
	}

	// Check nonce replay
	if !v.nonceStore.Add(env.Nonce) {
		return nil, VerificationResult{
			Reason:    "duplicate nonce (replay detected)",
			UserID:    env.UserID,
			Origin:    env.Origin,
			Timestamp: env.Timestamp,
		}
	}

	// Extract command (strip signing fields)
	command = stripSigningFields(raw)

	// Build canonical signed payload
	payload := SignedPayload{
		Command:   json.RawMessage(command),
		Timestamp: env.Timestamp,
		Nonce:     env.Nonce,
		UserID:    env.UserID,
		Origin:    env.Origin,
	}

	canonical, err := json.Marshal(payload)
	if err != nil {
		return nil, VerificationResult{Reason: fmt.Sprintf("failed to build canonical payload: %v", err)}
	}

	// Decode signature
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		sig, err = base64.RawStdEncoding.DecodeString(env.Signature)
		if err != nil {
			return nil, VerificationResult{Reason: "invalid signature encoding"}
		}
	}

	if !ed25519.Verify(v.pubKey, canonical, sig) {
		return nil, VerificationResult{
			Reason:    "signature verification failed",
			UserID:    env.UserID,
			Origin:    env.Origin,
			Timestamp: env.Timestamp,
		}
	}

	result.Valid = true
	return command, result
}

// stripSigningFields removes signature, timestamp, nonce, user_id, origin from the JSON.
func stripSigningFields(raw []byte) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	delete(m, "signature")
	delete(m, "timestamp")
	delete(m, "nonce")
	delete(m, "user_id")
	delete(m, "origin")
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

// Sign creates a signed message (for testing and SaaS use).
func Sign(privKey ed25519.PrivateKey, command []byte, timestamp int64, nonce, userID, origin string) ([]byte, error) {
	// Normalize command JSON (sorted keys) to match verification
	command = normalizeJSON(command)
	payload := SignedPayload{
		Command:   json.RawMessage(command),
		Timestamp: timestamp,
		Nonce:     nonce,
		UserID:    userID,
		Origin:    origin,
	}

	canonical, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	sig := ed25519.Sign(privKey, canonical)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(command, &m); err != nil {
		return nil, fmt.Errorf("unmarshal command: %w", err)
	}

	m["signature"] = mustMarshal(base64.StdEncoding.EncodeToString(sig))
	m["timestamp"] = mustMarshal(timestamp)
	m["nonce"] = mustMarshal(nonce)
	m["user_id"] = mustMarshal(userID)
	m["origin"] = mustMarshal(origin)

	return json.Marshal(m)
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// NonceStore tracks seen nonces with TTL-based expiration.
type NonceStore struct {
	mu     sync.Mutex
	nonces map[string]time.Time
	ttl    time.Duration
	lastGC time.Time
}

// NewNonceStore creates a nonce store with the given TTL.
func NewNonceStore(ttl time.Duration) *NonceStore {
	return &NonceStore{
		nonces: make(map[string]time.Time),
		ttl:    ttl,
		lastGC: time.Now(),
	}
}

// Add tries to add a nonce. Returns true if new, false if replay.
func (ns *NonceStore) Add(nonce string) bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	now := time.Now()
	if now.Sub(ns.lastGC) > ns.ttl {
		for k, t := range ns.nonces {
			if now.Sub(t) > ns.ttl {
				delete(ns.nonces, k)
			}
		}
		ns.lastGC = now
	}

	if _, exists := ns.nonces[nonce]; exists {
		return false
	}
	ns.nonces[nonce] = now
	return true
}

// normalizeJSON re-marshals JSON to get consistent key ordering.
func normalizeJSON(raw []byte) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}
