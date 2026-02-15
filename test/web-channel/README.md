# Web Channel Test Suite

## Overview

Comprehensive test suite for PicoClaw's Web Channel implementation, covering unit tests, integration tests, and manual browser testing.

## Test Structure

```
test/web-channel/
├── config.test.json          # Test configuration
├── client.html               # Browser test client
├── test-manual.sh           # Manual interactive test
├── test-automated.sh        # Automated CI/CD test
├── workspace/               # Test workspace
└── README.md                # This file
```

## Prerequisites

- Go 1.21+
- Built binary: `make build`
- API key (optional): `export ANTHROPIC_API_KEY=sk-...`
- Modern browser (Chrome/Firefox/Safari)

## Test Levels

### 1. Unit Tests (Fast, Isolated)

**Run:**
```bash
go test -v -cover ./pkg/channels -run TestWeb
```

**Coverage:** Tests WebChannel core functionality
- ✅ Authentication flow (no auth, with auth, wrong token)
- ✅ Rate limiting (per-session, global)
- ✅ Session expiry
- ✅ Token bucket algorithm
- ✅ HTTP handlers (index, status)

**Current Coverage:** 3.9% (handlers only, WebSocket not covered)

**Expected Output:**
```
=== RUN   TestWebChannel_Authentication_NoAuth
--- PASS: TestWebChannel_Authentication_NoAuth (0.00s)
...
PASS
coverage: 3.9% of statements
```

### 2. Automated Integration Tests

**Run:**
```bash
cd test/web-channel
./test-automated.sh
```

**Tests:**
1. HTTP Endpoints
   - GET / → 200 (index page)
   - GET /api/status → 200 (status JSON)
   - POST /auth (wrong token) → 401
   - POST /auth (correct token) → 200
   - GET /ws → 400 (upgrade required)

2. Cookie handling
   - Session cookie creation
   - HttpOnly, SameSite attributes

3. Unit tests pass

**Expected Output:**
```
========================================
PicoClaw Web Channel Automated Test
========================================

✓ Server started (PID: 12345)

Testing: GET /... PASS (HTTP 200)
Testing: GET /api/status (no auth)... PASS (HTTP 200)
...

========================================
Test Results
========================================
Passed: 6
Failed: 0
========================================

✓ All tests passed!
```

### 3. Manual Browser Tests

**Run:**
```bash
cd test/web-channel
./test-manual.sh
```

**What it does:**
1. Builds binary if needed
2. Creates test workspace
3. Starts PicoClaw server on http://127.0.0.1:8080
4. Opens `client.html` in browser
5. Streams server logs

**Test Cases:**

#### TC1: Authentication Flow
1. Open client in browser
2. Enter token: `test-secret-token-123`
3. Click "Authenticate"
4. ✅ Status → "Connected"
5. ✅ System message: "✓ Authentication successful"

#### TC2: WebSocket Connection
1. After authentication
2. ✅ Status → "Connected" (green)
3. ✅ Stats show uptime, sessions count
4. Check browser console: no WebSocket errors

#### TC3: Message Exchange
1. Type message: "hello"
2. Press Enter or click Send
3. ✅ User message appears (blue, right-aligned)
4. ✅ Agent response appears (gray, left-aligned)
5. ✅ Latency updates in stats

#### TC4: Rate Limiting (Per-Session)
1. Rapidly send 15 messages
2. ✅ After 10th message: rate limit warning
3. ✅ Messages 11-15 blocked
4. Wait 1 second
5. ✅ Can send again

#### TC5: Session Management
1. Get session count from `/api/status`
2. Open 2nd browser tab → authenticate
3. ✅ Session count increases
4. Close tab
5. ✅ Session count decreases (within 60s)

#### TC6: Reconnection
1. Stop server (Ctrl+C)
2. ✅ Status → "Disconnected" (red)
3. Restart server
4. Re-authenticate
5. ✅ Connection restored

#### TC7: Mobile Responsive (if implemented)
1. Open DevTools
2. Toggle device toolbar
3. Test on iPhone/Android viewport
4. ✅ UI adapts to small screen
5. ✅ All functions work

#### TC8: ARIA Accessibility (if implemented)
1. Use screen reader (VoiceOver/NVDA)
2. Navigate with Tab key
3. ✅ All controls announced
4. ✅ Form labels present
5. ✅ Status updates announced

## Test Configuration

**File:** `config.test.json`

```json
{
  "channels": {
    "web": {
      "enabled": true,
      "host": "127.0.0.1",
      "port": 8080,
      "auth_token": "test-secret-token-123",
      "session_max_age": 3600,
      "allow_from": ["u1"]
    }
  }
}
```

**Security Note:** Token is visible in config. In production, use:
```bash
export PICOCLAW_CHANNELS_WEB_AUTH_TOKEN=secret
```

## Troubleshooting

### Server won't start
```bash
# Check port availability
lsof -i :8080

# Check logs
tail -f ../../picoclaw.log

# Try different port
# Edit config.test.json: "port": 8081
```

### WebSocket connection fails
1. Check browser console for errors
2. Verify authentication succeeded (cookie set)
3. Check server logs: `tail -f picoclaw.log`
4. Test with curl:
   ```bash
   # Auth first
   curl -X POST http://127.0.0.1:8080/auth \
     -H "Content-Type: application/json" \
     -d '{"token":"test-secret-token-123"}' \
     -c cookies.txt
   
   # Check status
   curl http://127.0.0.1:8080/api/status -b cookies.txt
   ```

### Rate limiting too aggressive
Edit `/pkg/channels/web.go`:
```go
// Line 92-93
perSession sync.Map     // sessionID → *tokenBucket (10 msg/s)
global     *tokenBucket // 100 msg/s total

// Change to:
newTokenBucket(100, 100) // 100 msg/s per session
```

### No agent responses
1. Check `ANTHROPIC_API_KEY` is set
2. Verify provider config in `config.test.json`
3. Check agent loop is running: look for "Agent loop started" in logs

## CI/CD Integration

**GitHub Actions example:**
```yaml
- name: Test Web Channel
  run: |
    cd test/web-channel
    ./test-automated.sh
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

## Performance Benchmarks

**Expected metrics:**
- Connection setup: <100ms
- Message latency: <50ms
- Memory per session: <1MB
- Max concurrent sessions: 50
- Rate limit: 10 msg/s per session, 100 msg/s global

**Measure with:**
```bash
# Start server with profiling
go test -bench=. -cpuprofile=cpu.prof ./pkg/channels

# Analyze
go tool pprof cpu.prof
```

## Test Coverage Roadmap

Current: 3.9% → Target: >70%

**Missing tests:**
- [ ] WebSocket connection lifecycle
- [ ] Message serialization/deserialization
- [ ] Session cleanup on disconnect
- [ ] Concurrent connection handling
- [ ] Error recovery
- [ ] SSRF protection (isInternalHost)

**Add to `web_test.go`:**
```go
func TestWebChannel_WebSocketLifecycle(t *testing.T)
func TestWebChannel_ConcurrentConnections(t *testing.T)
func TestWebChannel_MessageSerialization(t *testing.T)
```

## Security Testing

**Manual checks:**
1. ✅ Cookies: HttpOnly, SameSite=Strict
2. ✅ Rate limiting works
3. ✅ Session expiry enforced
4. ⚠️ SSRF protection (needs implementation)
5. ⚠️ XSS prevention in messages

**Penetration testing:**
```bash
# Test rate limit bypass
for i in {1..20}; do
  curl -X POST http://127.0.0.1:8080/auth \
    -H "Content-Type: application/json" \
    -d '{"token":"test"}' &
done

# Test session hijacking
# 1. Get valid session cookie
# 2. Use in different browser/IP
# 3. Should work (cookie-based auth)
```

## Known Issues

1. **WebSocket not in unit tests** 
   - Coverage: 3.9% (only handlers)
   - Need: Mock WebSocket connections

2. **No SSRF protection yet**
   - Status: Part 0 security fixes pending
   - Risk: Agent could fetch internal URLs

3. **Mobile UI not implemented**
   - Plan approved, awaiting confirmation

4. **No ARIA landmarks**
   - Plan approved, awaiting confirmation

## References

- Implementation: [pkg/channels/web.go](../../pkg/channels/web.go)
- Unit tests: [pkg/channels/web_test.go](../../pkg/channels/web_test.go)
- Config: [pkg/config/config.go](../../pkg/config/config.go)
- Web Plan: [CLAUDE.md](../../CLAUDE.md) (search "Web Channel")
