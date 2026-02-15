package channels

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newTestWebChannel(authToken string) *WebChannel {
	cfg := config.WebConfig{
		Enabled:       true,
		Host:          "127.0.0.1",
		Port:          0,
		AuthToken:     authToken,
		SessionMaxAge: 3600,
	}
	messageBus := bus.NewMessageBus()
	ch, _ := NewWebChannel(cfg, messageBus)
	return ch
}

func TestWebChannel_Authentication_NoAuth(t *testing.T) {
	ch := newTestWebChannel("")

	// Without auth token configured, /api/status should be accessible
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	ch.startTime = time.Now()
	ch.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["auth_enabled"] != false {
		t.Error("expected auth_enabled=false")
	}
}

func TestWebChannel_Authentication_WithAuth_NoCookie(t *testing.T) {
	ch := newTestWebChannel("secret123")

	// With auth configured but no cookie, should return 401
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	ch.startTime = time.Now()
	ch.handleStatus(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWebChannel_Authentication_AuthFlow(t *testing.T) {
	ch := newTestWebChannel("secret123")

	// 1. POST /auth with correct token
	body := `{"token":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ch.handleAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("auth expected 200, got %d", w.Code)
	}

	var authResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&authResp)
	if authResp["ok"] != true {
		t.Error("expected ok=true")
	}

	// Check cookie was set
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "picoclaw_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected picoclaw_session cookie")
	}
	if !sessionCookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Error("cookie should have SameSite=Strict")
	}

	// 2. Use cookie to access /api/status
	req2 := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	ch.startTime = time.Now()
	ch.handleStatus(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("status with cookie expected 200, got %d", w2.Code)
	}
}

func TestWebChannel_Authentication_WrongToken(t *testing.T) {
	ch := newTestWebChannel("secret123")

	body := `{"token":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ch.handleAuth(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWebChannel_RateLimit_PerSession(t *testing.T) {
	rl := newRateLimiter()

	// 10 messages should pass
	for i := 0; i < 10; i++ {
		if !rl.allow("session1") {
			t.Fatalf("message %d should be allowed", i+1)
		}
	}

	// 11th should be rate limited
	if rl.allow("session1") {
		t.Error("11th message should be rate limited")
	}

	// Different session should still work
	if !rl.allow("session2") {
		t.Error("different session should be allowed")
	}
}

func TestWebChannel_RateLimit_Global(t *testing.T) {
	rl := newRateLimiter()

	// Exhaust global limit (100 msg/s)
	for i := 0; i < 100; i++ {
		sid := string(rune('a' + i%26))
		rl.allow(sid)
	}

	// Next should be rate limited globally
	if rl.allow("newSession") {
		t.Error("should hit global rate limit")
	}
}

func TestWebChannel_SessionExpiry(t *testing.T) {
	ch := newTestWebChannel("")
	ch.startTime = time.Now()

	// Create a session with very short maxAge
	sess := &wsSession{
		id:        "test-expired",
		createdAt: time.Now().Add(-2 * time.Hour),
		lastSeen:  time.Now().Add(-2 * time.Hour),
		maxAge:    1 * time.Hour,
		send:      make(chan []byte, 10),
	}

	ch.sessions.Store(sess.id, sess)
	ch.connCount.Add(1)

	// Verify session exists
	if _, ok := ch.sessions.Load("test-expired"); !ok {
		t.Fatal("session should exist")
	}

	// Simulate cleanup check (without conn, just test the logic)
	now := time.Now()
	ch.sessions.Range(func(key, value interface{}) bool {
		s := value.(*wsSession)
		if now.Sub(s.lastSeen) > s.maxAge {
			ch.sessions.Delete(key)
			ch.connCount.Add(-1)
		}
		return true
	})

	// Session should be removed
	if _, ok := ch.sessions.Load("test-expired"); ok {
		t.Error("expired session should be removed")
	}

	if ch.connCount.Load() != 0 {
		t.Errorf("connCount should be 0, got %d", ch.connCount.Load())
	}
}

func TestWebChannel_TokenBucket(t *testing.T) {
	tb := newTokenBucket(5, 5)

	// Use all tokens
	for i := 0; i < 5; i++ {
		if !tb.allow() {
			t.Fatalf("token %d should be available", i+1)
		}
	}

	// Should be empty
	if tb.allow() {
		t.Error("bucket should be empty")
	}

	// Wait for refill
	time.Sleep(210 * time.Millisecond) // ~1 token refilled at 5/s

	if !tb.allow() {
		t.Error("should have refilled 1 token")
	}
}

func TestWebChannel_IndexHandler(t *testing.T) {
	ch := newTestWebChannel("")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	ch.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %s", ct)
	}

	// Security headers
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options header")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options header")
	}
}

func TestWebChannel_StatusResponse(t *testing.T) {
	ch := newTestWebChannel("")
	ch.startTime = time.Now().Add(-60 * time.Second)

	status := ch.buildStatus()

	if _, ok := status["active_connections"]; !ok {
		t.Error("missing active_connections")
	}
	if _, ok := status["uptime_sec"]; !ok {
		t.Error("missing uptime_sec")
	}
	if status["auth_enabled"] != false {
		t.Error("expected auth_enabled=false")
	}

	uptimeSec := status["uptime_sec"].(int)
	if uptimeSec < 59 {
		t.Errorf("expected uptime >= 59s, got %d", uptimeSec)
	}
}
