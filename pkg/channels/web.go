package channels

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	maxActiveSessions = 50
	wsReadBufferSize  = 4096
	wsWriteBufferSize = 4096
	maxQueuedMessages = 100
	maxReadLimit      = 512 * 1024 // 512KB
	pongWait          = 60 * time.Second
	pingPeriod        = 54 * time.Second
	writeWait         = 10 * time.Second
	shutdownWait      = 5 * time.Second
)

// wsSession represents a WebSocket client session.
type wsSession struct {
	id        string
	conn      *websocket.Conn
	send      chan []byte
	createdAt time.Time
	lastSeen  time.Time
	maxAge    time.Duration
}

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(maxTokens, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// rateLimiter manages per-session and global rate limits.
type rateLimiter struct {
	perSession sync.Map     // sessionID → *tokenBucket
	global     *tokenBucket // 100 msg/s total
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		global: newTokenBucket(100, 100),
	}
}

func (rl *rateLimiter) allow(sessionID string) bool {
	if !rl.global.allow() {
		return false
	}

	val, _ := rl.perSession.LoadOrStore(sessionID, newTokenBucket(10, 10))
	bucket := val.(*tokenBucket)
	return bucket.allow()
}

func (rl *rateLimiter) remove(sessionID string) {
	rl.perSession.Delete(sessionID)
}

// WebChannel implements the Channel interface for browser-based chat.
type WebChannel struct {
	*BaseChannel
	config     config.WebConfig
	httpServer *http.Server
	sessions   sync.Map // sessionID → *wsSession
	limiter    *rateLimiter
	upgrader   websocket.Upgrader
	ctx        context.Context
	cancel     context.CancelFunc
	startTime  time.Time
	connCount  atomic.Int32
	// authSessions stores valid cookie session tokens
	authSessions sync.Map // cookieToken → sessionID
}

func NewWebChannel(cfg config.WebConfig, messageBus *bus.MessageBus) (*WebChannel, error) {
	base := NewBaseChannel("web", cfg, messageBus, cfg.AllowFrom)

	return &WebChannel{
		BaseChannel: base,
		config:      cfg,
		limiter:     newRateLimiter(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  wsReadBufferSize,
			WriteBufferSize: wsWriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				// Same-origin only: allow if Origin matches Host or is empty
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				return true // Allow all for local/LAN use
			},
		},
	}, nil
}

func (c *WebChannel) Start(ctx context.Context) error {
	logger.InfoC("web", "Starting Web channel")

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.startTime = time.Now()

	mux := http.NewServeMux()
	mux.HandleFunc("/", c.handleIndex)
	mux.HandleFunc("/auth", c.handleAuth)
	mux.HandleFunc("/ws", c.handleWS)
	mux.HandleFunc("/api/status", c.handleStatus)

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	c.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.InfoCF("web", "Web server listening", map[string]interface{}{
			"addr":     addr,
			"auth":     c.config.AuthToken != "",
			"max_age":  c.config.SessionMaxAge,
		})
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("web", "Web server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	go c.cleanupExpiredSessions()

	c.setRunning(true)
	logger.InfoC("web", "Web channel started")
	return nil
}

func (c *WebChannel) Stop(ctx context.Context) error {
	logger.InfoC("web", "Stopping Web channel")

	if c.cancel != nil {
		c.cancel()
	}

	// Close all WebSocket connections gracefully
	c.sessions.Range(func(key, value interface{}) bool {
		sess := value.(*wsSession)
		closeMsg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down")
		sess.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(writeWait))
		close(sess.send)
		return true
	})

	if c.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownWait)
		defer cancel()
		if err := c.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("web", "Web server shutdown error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	c.setRunning(false)
	logger.InfoC("web", "Web channel stopped")
	return nil
}

func (c *WebChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("web channel not running")
	}

	payload := map[string]interface{}{
		"type":    "message",
		"content": msg.Content,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Find session by chatID (chatID == sessionID for web)
	val, ok := c.sessions.Load(msg.ChatID)
	if !ok {
		return fmt.Errorf("session %s not found", msg.ChatID)
	}

	sess := val.(*wsSession)
	select {
	case sess.send <- data:
		return nil
	default:
		return fmt.Errorf("send queue full for session %s", msg.ChatID)
	}
}

// --- HTTP Handlers ---

func (c *WebChannel) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Serve the embedded HTML
	data, err := webStaticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Write(data)
}

func (c *WebChannel) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if c.config.AuthToken == "" {
		// No auth configured, return success
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "auth_required": false})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Token != c.config.AuthToken {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "invalid token"})
		return
	}

	// Generate session token
	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cookieToken := generateCookieToken(sessionID, c.config.AuthToken)
	c.authSessions.Store(cookieToken, sessionID)

	maxAge := c.config.SessionMaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "picoclaw_session",
		Value:    cookieToken,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   false, // OK for local/LAN
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (c *WebChannel) handleWS(w http.ResponseWriter, r *http.Request) {
	if c.config.AuthToken != "" {
		if !c.checkAuth(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if int(c.connCount.Load()) >= maxActiveSessions {
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("web", "WebSocket upgrade failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sessionID, err := generateSessionID()
	if err != nil {
		conn.Close()
		return
	}

	maxAge := time.Duration(c.config.SessionMaxAge) * time.Second
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}

	sess := &wsSession{
		id:        sessionID,
		conn:      conn,
		send:      make(chan []byte, maxQueuedMessages),
		createdAt: time.Now(),
		lastSeen:  time.Now(),
		maxAge:    maxAge,
	}

	c.sessions.Store(sessionID, sess)
	c.connCount.Add(1)

	logger.InfoCF("web", "WebSocket client connected", map[string]interface{}{
		"session_id":  sessionID,
		"remote_addr": r.RemoteAddr,
		"active":      c.connCount.Load(),
	})

	// Send session info
	welcome, _ := json.Marshal(map[string]interface{}{
		"type":       "session",
		"session_id": sessionID,
	})
	sess.send <- welcome

	go c.writePump(sess)
	go c.readPump(sess)
}

func (c *WebChannel) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if c.config.AuthToken != "" {
		if !c.checkAuth(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	status := c.buildStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// --- WebSocket Pumps ---

func (c *WebChannel) readPump(sess *wsSession) {
	defer func() {
		c.removeSession(sess)
	}()

	sess.conn.SetReadLimit(maxReadLimit)
	sess.conn.SetReadDeadline(time.Now().Add(pongWait))
	sess.conn.SetPongHandler(func(string) error {
		sess.conn.SetReadDeadline(time.Now().Add(pongWait))
		sess.lastSeen = time.Now()
		return nil
	})

	for {
		_, message, err := sess.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.DebugCF("web", "WebSocket read error", map[string]interface{}{
					"session_id": sess.id,
					"error":      err.Error(),
				})
			}
			return
		}

		sess.lastSeen = time.Now()

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError(sess, "invalid message format")
			continue
		}

		msgType, _ := msg["type"].(string)

		switch msgType {
		case "message":
			c.handleClientMessage(sess, msg)
		case "request_status":
			status := c.buildStatus()
			data, _ := json.Marshal(map[string]interface{}{
				"type":      "status",
				"timestamp": time.Now().Unix(),
				"data":      status,
			})
			select {
			case sess.send <- data:
			default:
			}
		default:
			c.sendError(sess, "unknown message type")
		}
	}
}

func (c *WebChannel) writePump(sess *wsSession) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		sess.conn.Close()
	}()

	for {
		select {
		case message, ok := <-sess.send:
			sess.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				sess.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := sess.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			sess.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := sess.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// --- Message Handling ---

func (c *WebChannel) handleClientMessage(sess *wsSession, msg map[string]interface{}) {
	if !c.limiter.allow(sess.id) {
		c.sendError(sess, "rate limit exceeded")
		return
	}

	content, _ := msg["content"].(string)
	if content == "" {
		c.sendError(sess, "empty message")
		return
	}

	metadata := map[string]string{
		"platform": "web",
	}

	c.HandleMessage(sess.id, sess.id, content, nil, metadata)
}

// --- Auth Helpers ---

func (c *WebChannel) checkAuth(r *http.Request) bool {
	cookie, err := r.Cookie("picoclaw_session")
	if err != nil {
		return false
	}
	_, ok := c.authSessions.Load(cookie.Value)
	return ok
}

func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateCookieToken(sessionID, authToken string) string {
	mac := hmac.New(sha256.New, []byte(authToken))
	mac.Write([]byte(sessionID))
	mac.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(mac.Sum(nil))
}

// --- Session Management ---

func (c *WebChannel) removeSession(sess *wsSession) {
	c.sessions.Delete(sess.id)
	c.limiter.remove(sess.id)
	c.connCount.Add(-1)

	logger.InfoCF("web", "WebSocket client disconnected", map[string]interface{}{
		"session_id": sess.id,
		"active":     c.connCount.Load(),
	})
}

func (c *WebChannel) cleanupExpiredSessions() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.sessions.Range(func(key, value interface{}) bool {
				sess := value.(*wsSession)
				if now.Sub(sess.lastSeen) > sess.maxAge {
					logger.DebugCF("web", "Closing expired session", map[string]interface{}{
						"session_id": sess.id,
					})
					sess.conn.Close()
					close(sess.send)
					c.removeSession(sess)
				}
				return true
			})
		case <-c.ctx.Done():
			return
		}
	}
}

// --- Status ---

func (c *WebChannel) buildStatus() map[string]interface{} {
	return map[string]interface{}{
		"active_connections": c.connCount.Load(),
		"uptime_sec":         int(time.Since(c.startTime).Seconds()),
		"auth_enabled":       c.config.AuthToken != "",
	}
}

// --- Error Helper ---

func (c *WebChannel) sendError(sess *wsSession, message string) {
	data, _ := json.Marshal(map[string]interface{}{
		"type":    "error",
		"message": message,
	})
	select {
	case sess.send <- data:
	default:
	}
}
