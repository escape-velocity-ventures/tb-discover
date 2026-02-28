package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func generateKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func signCommand(t *testing.T, priv ed25519.PrivateKey, command []byte, ts int64, nonce, userID, origin string) []byte {
	t.Helper()
	signed, err := Sign(priv, command, ts, nonce, userID, origin)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestVerifyValidSignature(t *testing.T) {
	pub, priv := generateKeyPair(t)
	v := NewVerifier(pub)

	cmd := []byte(`{"type":"session.open","sessionId":"abc123"}`)
	now := time.Now().Unix()
	signed := signCommand(t, priv, cmd, now, "nonce1", "user1", "saas")

	command, result := v.Verify(signed)
	if !result.Valid {
		t.Fatalf("expected valid, got: %s", result.Reason)
	}
	if result.UserID != "user1" {
		t.Errorf("expected user1, got %s", result.UserID)
	}

	// Command should not contain signing fields
	var m map[string]json.RawMessage
	json.Unmarshal(command, &m)
	for _, field := range []string{"signature", "timestamp", "nonce", "user_id", "origin"} {
		if _, ok := m[field]; ok {
			t.Errorf("command should not contain %s", field)
		}
	}
	if _, ok := m["type"]; !ok {
		t.Error("command should contain type")
	}
}

func TestVerifyExpiredTimestamp(t *testing.T) {
	pub, priv := generateKeyPair(t)
	v := NewVerifier(pub)

	cmd := []byte(`{"type":"session.open"}`)
	old := time.Now().Add(-60 * time.Second).Unix()
	signed := signCommand(t, priv, cmd, old, "nonce2", "user1", "saas")

	_, result := v.Verify(signed)
	if result.Valid {
		t.Fatal("expected rejection for old timestamp")
	}
	if result.Reason == "" {
		t.Fatal("expected reason")
	}
}

func TestVerifyFutureTimestamp(t *testing.T) {
	pub, priv := generateKeyPair(t)
	v := NewVerifier(pub)

	cmd := []byte(`{"type":"session.open"}`)
	future := time.Now().Add(60 * time.Second).Unix()
	signed := signCommand(t, priv, cmd, future, "nonce3", "user1", "saas")

	_, result := v.Verify(signed)
	if result.Valid {
		t.Fatal("expected rejection for future timestamp")
	}
}

func TestVerifyReplayNonce(t *testing.T) {
	pub, priv := generateKeyPair(t)
	v := NewVerifier(pub)

	cmd := []byte(`{"type":"session.open"}`)
	now := time.Now().Unix()
	signed := signCommand(t, priv, cmd, now, "nonce-replay", "user1", "saas")

	_, result := v.Verify(signed)
	if !result.Valid {
		t.Fatalf("first should be valid: %s", result.Reason)
	}

	// Same nonce again
	signed2 := signCommand(t, priv, cmd, now, "nonce-replay", "user1", "saas")
	_, result2 := v.Verify(signed2)
	if result2.Valid {
		t.Fatal("expected replay rejection")
	}
	if result2.Reason != "duplicate nonce (replay detected)" {
		t.Errorf("unexpected reason: %s", result2.Reason)
	}
}

func TestVerifyWrongKey(t *testing.T) {
	pub, _ := generateKeyPair(t)
	_, otherPriv := generateKeyPair(t)
	v := NewVerifier(pub)

	cmd := []byte(`{"type":"session.open"}`)
	now := time.Now().Unix()
	signed := signCommand(t, otherPriv, cmd, now, "nonce4", "user1", "saas")

	_, result := v.Verify(signed)
	if result.Valid {
		t.Fatal("expected rejection for wrong key")
	}
	if result.Reason != "signature verification failed" {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
}

func TestVerifyMissingSignature(t *testing.T) {
	pub, _ := generateKeyPair(t)
	v := NewVerifier(pub)

	raw := []byte(`{"type":"session.open","nonce":"n","timestamp":1}`)
	_, result := v.Verify(raw)
	if result.Valid {
		t.Fatal("expected rejection for missing signature")
	}
}

func TestVerifyMissingNonce(t *testing.T) {
	pub, _ := generateKeyPair(t)
	v := NewVerifier(pub)

	raw := []byte(`{"type":"session.open","signature":"abc","timestamp":1}`)
	_, result := v.Verify(raw)
	if result.Valid {
		t.Fatal("expected rejection for missing nonce")
	}
}

func TestVerifyTamperedMessage(t *testing.T) {
	pub, priv := generateKeyPair(t)
	v := NewVerifier(pub)

	cmd := []byte(`{"type":"session.open","sessionId":"legit"}`)
	now := time.Now().Unix()
	signed := signCommand(t, priv, cmd, now, "nonce5", "user1", "saas")

	// Tamper with the message
	var m map[string]json.RawMessage
	json.Unmarshal(signed, &m)
	m["sessionId"] = json.RawMessage(`"evil"`)
	tampered, _ := json.Marshal(m)

	_, result := v.Verify(tampered)
	if result.Valid {
		t.Fatal("expected rejection for tampered message")
	}
}

func TestParsePublicKeyHex(t *testing.T) {
	pub, _ := generateKeyPair(t)
	hexStr := hex.EncodeToString(pub)

	parsed, err := ParsePublicKey(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(parsed) {
		t.Fatal("parsed key doesn't match")
	}
}

func TestParsePublicKeyBase64(t *testing.T) {
	pub, _ := generateKeyPair(t)
	b64Str := base64.StdEncoding.EncodeToString(pub)

	parsed, err := ParsePublicKey(b64Str)
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Equal(parsed) {
		t.Fatal("parsed key doesn't match")
	}
}

func TestParsePublicKeyEmpty(t *testing.T) {
	_, err := ParsePublicKey("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestParsePublicKeyInvalid(t *testing.T) {
	_, err := ParsePublicKey("not-a-key")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestNonceStoreGC(t *testing.T) {
	ns := NewNonceStore(10 * time.Millisecond)
	ns.Add("old-nonce")
	time.Sleep(20 * time.Millisecond)
	// Should be able to add again after TTL
	if !ns.Add("new-nonce") {
		t.Fatal("should accept new nonce")
	}
	// Force GC by adding after TTL
	time.Sleep(20 * time.Millisecond)
	if !ns.Add("old-nonce") {
		t.Fatal("old nonce should have been GC'd")
	}
}

func TestNonceStoreConcurrent(t *testing.T) {
	ns := NewNonceStore(time.Minute)
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(n int) {
			ns.Add("concurrent-nonce")
			done <- true
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestSignAndVerifyRoundTrip(t *testing.T) {
	pub, priv := generateKeyPair(t)
	v := NewVerifier(pub)

	commands := []string{
		`{"type":"session.open","sessionId":"s1","hostId":"h1"}`,
		`{"type":"pty.input","sessionId":"s1","data":"ls\n"}`,
		`{"type":"pty.resize","sessionId":"s1","cols":120,"rows":40}`,
		`{"type":"session.close","sessionId":"s1"}`,
	}

	for i, cmdStr := range commands {
		now := time.Now().Unix()
		nonce := "roundtrip-" + string(rune('a'+i))
		signed := signCommand(t, priv, []byte(cmdStr), now, nonce, "admin", "saas-api")
		cmd, result := v.Verify(signed)
		if !result.Valid {
			t.Errorf("command %d: expected valid, got %s", i, result.Reason)
			continue
		}
		// Verify command content preserved
		var orig, parsed map[string]json.RawMessage
		json.Unmarshal([]byte(cmdStr), &orig)
		json.Unmarshal(cmd, &parsed)
		for k := range orig {
			if string(orig[k]) != string(parsed[k]) {
				t.Errorf("command %d: field %s mismatch: %s vs %s", i, k, orig[k], parsed[k])
			}
		}
	}
}
