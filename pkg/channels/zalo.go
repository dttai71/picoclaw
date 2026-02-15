package channels

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	zaloOAAPIBase       = "https://openapi.zalo.me/v3.0/oa"
	zaloSendEndpoint    = zaloOAAPIBase + "/message/cs"
	zaloOAuthBase       = "https://oauth.zaloapp.com/v4"
	zaloOAuthTokenURL   = zaloOAuthBase + "/oa/access_token"
	zaloOAuthAuthorize  = "https://oauth.zaloapp.com/v4/oa/permission"
	zaloTokenFileName   = "zalo_tokens.json"
	zaloRefreshInterval = 5 * time.Minute
	zaloRefreshBuffer   = 30 * time.Minute
	zaloSendInterval    = 600 * time.Millisecond // ~100 msg/min
)

// zaloTokens holds the OAuth 2.0 token pair for Zalo OA.
type zaloTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// ZaloChannel implements the Channel interface for Zalo Official Account
// using the Zalo OA API v3.0 with HTTP webhook for receiving messages
// and REST API for sending messages.
type ZaloChannel struct {
	*BaseChannel
	config     config.ZaloConfig
	httpServer *http.Server
	tokens     *zaloTokens
	tokenMu    sync.RWMutex
	sendTicker chan time.Time // simple rate limiter for Send()
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewZaloChannel creates a new Zalo channel instance.
func NewZaloChannel(cfg config.ZaloConfig, messageBus *bus.MessageBus) (*ZaloChannel, error) {
	if cfg.AppID == "" || cfg.OASecretKey == "" {
		return nil, fmt.Errorf("zalo app_id and oa_secret_key are required")
	}

	base := NewBaseChannel("zalo", cfg, messageBus, cfg.AllowFrom)

	return &ZaloChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Start launches the HTTP webhook server and token refresh loop.
func (c *ZaloChannel) Start(ctx context.Context) error {
	logger.InfoC("zalo", "Starting Zalo channel (Webhook Mode)")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Initialize send rate limiter (~100 msg/min)
	c.sendTicker = make(chan time.Time, 1)
	c.sendTicker <- time.Now() // allow first send immediately
	go func() {
		ticker := time.NewTicker(zaloSendInterval)
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case t := <-ticker.C:
				select {
				case c.sendTicker <- t:
				default: // drop if buffer full
				}
			}
		}
	}()

	// Load stored tokens
	if err := c.loadTokens(); err != nil {
		logger.WarnCF("zalo", "No stored tokens, OAuth required", map[string]interface{}{
			"error": err.Error(),
		})
		logger.InfoC("zalo", "Run: picoclaw auth login --provider zalo")
	} else {
		logger.InfoC("zalo", "Tokens loaded successfully")
		// Start background token refresh
		go c.refreshTokenLoop()
	}

	mux := http.NewServeMux()
	path := c.config.WebhookPath
	if path == "" {
		path = "/webhook/zalo"
	}
	mux.HandleFunc(path, c.webhookHandler)

	addr := fmt.Sprintf("%s:%d", c.config.WebhookHost, c.config.WebhookPort)
	c.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		logger.InfoCF("zalo", "Zalo webhook server listening", map[string]interface{}{
			"addr": addr,
			"path": path,
		})
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("zalo", "Webhook server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	c.setRunning(true)
	logger.InfoC("zalo", "Zalo channel enabled successfully")
	return nil
}

// Stop gracefully shuts down the HTTP server and token refresh.
func (c *ZaloChannel) Stop(ctx context.Context) error {
	logger.InfoC("zalo", "Stopping Zalo channel")

	if c.cancel != nil {
		c.cancel()
	}

	if c.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("zalo", "Webhook server shutdown error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	c.setRunning(false)
	logger.InfoC("zalo", "Zalo channel stopped")
	return nil
}

// webhookHandler handles incoming Zalo webhook requests.
func (c *ZaloChannel) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorCF("zalo", "Failed to read request body", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-ZaloOA-Signature")
	if !c.verifySignature(body, signature) {
		logger.WarnC("zalo", "Invalid webhook signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var event zaloWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		logger.ErrorCF("zalo", "Failed to parse webhook payload", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Return 200 immediately, process event asynchronously
	w.WriteHeader(http.StatusOK)

	go c.processEvent(event)
}

// verifySignature validates the X-ZaloOA-Signature using HMAC-SHA256.
func (c *ZaloChannel) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(c.config.OASecretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// Zalo webhook event types
type zaloWebhookEvent struct {
	AppID     string          `json:"app_id"`
	Sender    zaloSender      `json:"sender"`
	Recipient zaloRecipient   `json:"recipient"`
	EventName string          `json:"event_name"`
	Message   json.RawMessage `json:"message"`
	Timestamp string          `json:"timestamp"`
}

type zaloSender struct {
	ID string `json:"id"`
}

type zaloRecipient struct {
	ID string `json:"id"`
}

type zaloMessage struct {
	MsgID string `json:"msg_id"`
	Text  string `json:"text"`
}

func (c *ZaloChannel) processEvent(event zaloWebhookEvent) {
	if event.EventName != "user_send_text" {
		logger.DebugCF("zalo", "Ignoring non-text event", map[string]interface{}{
			"event_name": event.EventName,
		})
		return
	}

	senderID := event.Sender.ID
	chatID := senderID // Zalo OA is 1:1 only

	var msg zaloMessage
	if err := json.Unmarshal(event.Message, &msg); err != nil {
		logger.ErrorCF("zalo", "Failed to parse message", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	content := strings.TrimSpace(msg.Text)
	if content == "" {
		return
	}

	metadata := map[string]string{
		"platform":   "zalo",
		"message_id": msg.MsgID,
	}

	logger.DebugCF("zalo", "Received message", map[string]interface{}{
		"sender_id": senderID,
		"chat_id":   chatID,
		"preview":   utils.Truncate(content, 50),
	})

	c.HandleMessage(senderID, chatID, content, nil, metadata)
}

// Send sends a text message to a Zalo user via the OA API.
func (c *ZaloChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("zalo channel not running")
	}

	// Wait for rate limiter
	select {
	case <-c.sendTicker:
	case <-ctx.Done():
		return ctx.Err()
	}

	c.tokenMu.RLock()
	tokens := c.tokens
	c.tokenMu.RUnlock()

	if tokens == nil || tokens.AccessToken == "" {
		return fmt.Errorf("no stored tokens, OAuth required")
	}

	// Proactively refresh if token is near expiry
	if !tokens.ExpiresAt.IsZero() && time.Until(tokens.ExpiresAt) < zaloRefreshBuffer {
		if err := c.refreshToken(); err != nil {
			logger.WarnCF("zalo", "Token refresh failed before send", map[string]interface{}{
				"error": err.Error(),
			})
		}
		c.tokenMu.RLock()
		tokens = c.tokens
		c.tokenMu.RUnlock()
	}

	payload := map[string]interface{}{
		"recipient": map[string]string{
			"user_id": msg.ChatID,
		},
		"message": map[string]string{
			"text": msg.Content,
		},
	}

	return c.callAPI(ctx, zaloSendEndpoint, payload, tokens.AccessToken)
}

// callAPI makes an authenticated POST request to the Zalo OA API.
func (c *ZaloChannel) callAPI(ctx context.Context, endpoint string, payload interface{}, accessToken string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		logger.WarnCF("zalo", "Rate limited by Zalo API", map[string]interface{}{
			"retry_after": resp.Header.Get("Retry-After"),
		})
		return fmt.Errorf("zalo API rate limited (429)")
	}

	if resp.StatusCode == http.StatusUnauthorized {
		logger.WarnC("zalo", "Access token rejected (401), clearing tokens")
		c.tokenMu.Lock()
		c.tokens = nil
		c.tokenMu.Unlock()
		return fmt.Errorf("zalo API unauthorized (401), re-run: picoclaw auth login --provider zalo")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("zalo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Check Zalo API-level error in response body
	var apiResp struct {
		Error int    `json:"error"`
		Msg   string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err == nil && apiResp.Error != 0 {
		return fmt.Errorf("zalo API error (code %d): %s", apiResp.Error, apiResp.Msg)
	}

	return nil
}

// --- OAuth 2.0 Token Management ---

// tokenFilePath returns the path to the Zalo token storage file.
func tokenFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", zaloTokenFileName)
}

// loadTokens reads tokens from ~/.picoclaw/zalo_tokens.json.
func (c *ZaloChannel) loadTokens() error {
	path := tokenFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var tokens zaloTokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	if tokens.AccessToken == "" {
		return fmt.Errorf("empty access token in file")
	}

	c.tokenMu.Lock()
	c.tokens = &tokens
	c.tokenMu.Unlock()

	return nil
}

// saveTokens writes tokens to ~/.picoclaw/zalo_tokens.json with atomic write and 0600 permissions.
func (c *ZaloChannel) saveTokens(tokens *zaloTokens) error {
	path := tokenFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp token file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename token file: %w", err)
	}

	c.tokenMu.Lock()
	c.tokens = tokens
	c.tokenMu.Unlock()

	return nil
}

// refreshTokenLoop runs in the background and refreshes the access token
// proactively when it's within 30 minutes of expiry. Checks every 5 minutes.
func (c *ZaloChannel) refreshTokenLoop() {
	ticker := time.NewTicker(zaloRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.tokenMu.RLock()
			tokens := c.tokens
			c.tokenMu.RUnlock()

			if tokens == nil {
				continue
			}

			if tokens.ExpiresAt.IsZero() || time.Until(tokens.ExpiresAt) > zaloRefreshBuffer {
				continue
			}

			logger.InfoC("zalo", "Access token expiring soon, refreshing")
			if err := c.refreshToken(); err != nil {
				logger.ErrorCF("zalo", "Token refresh failed", map[string]interface{}{
					"error": err.Error(),
				})
			} else {
				logger.InfoC("zalo", "Token refreshed successfully")
			}
		}
	}
}

// refreshToken exchanges the refresh token for a new access token.
func (c *ZaloChannel) refreshToken() error {
	c.tokenMu.RLock()
	tokens := c.tokens
	c.tokenMu.RUnlock()

	if tokens == nil || tokens.RefreshToken == "" {
		return fmt.Errorf("no refresh token available, re-run: picoclaw auth login --provider zalo")
	}

	data := url.Values{
		"refresh_token": {tokens.RefreshToken},
		"app_id":        {c.config.AppID},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequest(http.MethodPost, zaloOAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", c.config.AppSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        int    `json:"error"`
		Msg          string `json:"message"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	if tokenResp.Error != 0 {
		return fmt.Errorf("zalo OAuth error (code %d): %s", tokenResp.Error, tokenResp.Msg)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("no access token in refresh response")
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	newTokens := &zaloTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}

	return c.saveTokens(newTokens)
}

// ExchangeCodeForToken exchanges an authorization code for an access token.
// Used by the CLI auth login flow.
func (c *ZaloChannel) ExchangeCodeForToken(code string) error {
	data := url.Values{
		"code":       {code},
		"app_id":     {c.config.AppID},
		"grant_type": {"authorization_code"},
	}

	req, err := http.NewRequest(http.MethodPost, zaloOAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", c.config.AppSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        int    `json:"error"`
		Msg          string `json:"message"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != 0 {
		return fmt.Errorf("zalo OAuth error (code %d): %s", tokenResp.Error, tokenResp.Msg)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("no access token in response")
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	tokens := &zaloTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}

	return c.saveTokens(tokens)
}

// ZaloOAuthLogin performs the OAuth 2.0 browser login flow for Zalo OA.
// It starts a temporary HTTP server, opens the browser for authorization,
// waits for the callback with the authorization code, then exchanges it.
func ZaloOAuthLogin(cfg config.ZaloConfig) error {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return fmt.Errorf("zalo app_id and app_secret are required for OAuth login")
	}

	ch := &ZaloChannel{config: cfg}

	callbackPort := 18793
	redirectURI := fmt.Sprintf("http://localhost:%d/auth/zalo/callback", callbackPort)

	authURL := fmt.Sprintf("%s?app_id=%s&redirect_uri=%s",
		zaloOAuthAuthorize,
		url.QueryEscape(cfg.AppID),
		url.QueryEscape(redirectURI),
	)

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/zalo/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			resultCh <- callbackResult{err: fmt.Errorf("oauth error: %s", errMsg)}
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Zalo OAuth successful!</h2><p>You can close this window.</p></body></html>")
		resultCh <- callbackResult{code: code}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", callbackPort))
	if err != nil {
		return fmt.Errorf("failed to start callback server on port %d: %w", callbackPort, err)
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	fmt.Printf("Open this URL to authorize Zalo OA:\n\n%s\n\n", authURL)

	if err := zaloOpenBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically.\nPlease open this URL manually:\n\n%s\n\n", authURL)
	}

	fmt.Println("Waiting for authorization...")

	select {
	case result := <-resultCh:
		if result.err != nil {
			return fmt.Errorf("authorization failed: %w", result.err)
		}
		if err := ch.ExchangeCodeForToken(result.code); err != nil {
			return fmt.Errorf("token exchange failed: %w", err)
		}
		fmt.Println("OAuth successful")
		fmt.Printf("Tokens saved to %s\n", tokenFilePath())
		return nil
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authorization timed out after 5 minutes")
	}
}

func zaloOpenBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
