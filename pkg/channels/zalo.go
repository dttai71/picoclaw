package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	zaloBotAPIBase     = "https://bot-api.zapps.me/bot"
	zaloPollingTimeout = 30   // seconds
	zaloMaxChunkSize   = 2000 // characters
)

// ZaloChannel implements the Channel interface for Zalo Bot Platform API.
// Supports both long-polling (default) and webhook modes.
type ZaloChannel struct {
	*BaseChannel
	config     config.ZaloConfig
	apiBase    string // "https://bot-api.zapps.me/bot{token}"
	httpServer *http.Server
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
	lastUpdate int64
	updateMu   sync.Mutex
}

// NewZaloChannel creates a new Zalo Bot Platform channel instance.
func NewZaloChannel(cfg config.ZaloConfig, messageBus *bus.MessageBus) (*ZaloChannel, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("zalo bot token is required")
	}
	if !strings.Contains(cfg.Token, ":") {
		return nil, fmt.Errorf("invalid zalo bot token format (expected id:secret)")
	}

	base := NewBaseChannel("zalo", cfg, messageBus, cfg.AllowFrom)

	return &ZaloChannel{
		BaseChannel: base,
		config:      cfg,
		apiBase:     fmt.Sprintf("%s%s", zaloBotAPIBase, cfg.Token),
		httpClient: &http.Client{
			Timeout: 35 * time.Second, // slightly longer than polling timeout
		},
	}, nil
}

// Start launches the channel in either polling or webhook mode.
func (c *ZaloChannel) Start(ctx context.Context) error {
	logger.InfoC("zalo", "Starting Zalo Bot Platform channel")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Validate token by calling getMe
	me, err := c.getMe()
	if err != nil {
		return fmt.Errorf("failed to validate bot token: %w", err)
	}

	// Extract bot info from getMe response
	botName, _ := me["display_name"].(string)
	botID, _ := me["id"].(string)

	logger.InfoCF("zalo", "Bot connected", map[string]interface{}{
		"bot_name": botName,
		"bot_id":   botID,
		"mode":     c.config.Mode,
	})

	// Start appropriate mode
	if c.config.Mode == "webhook" {
		go c.startWebhookServer()
	} else {
		go c.pollingLoop()
	}

	c.setRunning(true)
	logger.InfoC("zalo", "Zalo channel started")
	return nil
}

// Stop gracefully shuts down the channel.
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

// Send sends a message to a Zalo user/group.
func (c *ZaloChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("zalo channel not running")
	}

	// Chunk long messages
	chunks := chunkText(msg.Content, zaloMaxChunkSize)
	for i, chunk := range chunks {
		if i > 0 {
			time.Sleep(500 * time.Millisecond) // rate limit between chunks
		}

		payload := map[string]interface{}{
			"chat_id": msg.ChatID,
			"text":    chunk,
		}

		if err := c.callAPI("sendMessage", payload, nil); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

// --- Bot API Methods ---

// zaloBotAPIResponse is the envelope for all Bot Platform API responses.
type zaloBotAPIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
	// Alternative format: {error, message, data}
	Error   int             `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (c *ZaloChannel) getMe() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.callAPI("getMe", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ZaloChannel) getUpdates(offset int64, timeout int) ([]map[string]interface{}, error) {
	payload := map[string]interface{}{
		"offset":  offset,
		"timeout": timeout,
	}

	// The API may return a single update object or an array of updates.
	// Try array first, then fall back to single object.
	var raw json.RawMessage
	if err := c.callAPI("getUpdates", payload, &raw); err != nil {
		return nil, err
	}

	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	// Try as array first
	var updates []map[string]interface{}
	if err := json.Unmarshal(raw, &updates); err == nil {
		return updates, nil
	}

	// Fall back to single object
	var single map[string]interface{}
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, fmt.Errorf("unmarshal updates: %w", err)
	}
	return []map[string]interface{}{single}, nil
}

// callAPI sends a POST to the Bot Platform API endpoint.
// It handles the response envelope ({ok, result} or {error, message, data})
// and unmarshals the inner result/data into the provided result parameter.
func (c *ZaloChannel) callAPI(method string, payload, result interface{}) error {
	url := fmt.Sprintf("%s/%s", c.apiBase, method)

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(c.ctx, "POST", url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusRequestTimeout {
		// 408 = no updates, not an error
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api error: %d %s", resp.StatusCode, string(respBody))
	}

	// Parse the API envelope
	var envelope zaloBotAPIResponse
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}

	// Check for errors in either format
	// 408 = long-polling timeout (no updates), not a real error
	if envelope.ErrorCode != 0 && envelope.ErrorCode != 408 {
		return fmt.Errorf("api error (code %d): %s", envelope.ErrorCode, envelope.Description)
	}
	if envelope.Error != 0 && envelope.Error != 408 {
		return fmt.Errorf("api error (code %d): %s", envelope.Error, envelope.Message)
	}

	// Unmarshal the inner result if requested
	if result != nil {
		// Try "result" field first (standard format), then "data" (alternative)
		inner := envelope.Result
		if len(inner) == 0 || string(inner) == "null" {
			inner = envelope.Data
		}
		if len(inner) > 0 && string(inner) != "null" {
			logger.DebugCF("zalo", "API raw response", map[string]interface{}{
				"method": method,
				"raw":    string(inner),
			})
			if err := json.Unmarshal(inner, result); err != nil {
				return fmt.Errorf("unmarshal result (raw=%s): %w", string(inner), err)
			}
		}
	}

	return nil
}

// --- Polling Mode ---

func (c *ZaloChannel) pollingLoop() {
	logger.InfoC("zalo", "Starting long-polling mode")

	backoff := time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.updateMu.Lock()
		offset := c.lastUpdate + 1
		c.updateMu.Unlock()

		updates, err := c.getUpdates(offset, zaloPollingTimeout)
		if err != nil {
			logger.ErrorCF("zalo", "Polling error", map[string]interface{}{
				"error": err.Error(),
			})
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on success
		backoff = time.Second

		for _, update := range updates {
			c.processUpdate(update)

			// Track offset using update_id or message.date
			var ts int64
			if uid, ok := update["update_id"].(float64); ok {
				ts = int64(uid)
			} else if msg, ok := update["message"].(map[string]interface{}); ok {
				if date, ok := msg["date"].(float64); ok {
					ts = int64(date)
				}
			}
			if ts > 0 {
				c.updateMu.Lock()
				if ts > c.lastUpdate {
					c.lastUpdate = ts
				}
				c.updateMu.Unlock()
			}
		}
	}
}

// --- Webhook Mode ---

func (c *ZaloChannel) startWebhookServer() {
	mux := http.NewServeMux()
	mux.HandleFunc(c.config.WebhookPath, c.webhookHandler)

	addr := fmt.Sprintf("%s:%d", c.config.WebhookHost, c.config.WebhookPort)
	c.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.InfoCF("zalo", "Webhook server listening", map[string]interface{}{
		"addr": addr,
		"path": c.config.WebhookPath,
	})

	if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.ErrorCF("zalo", "Webhook server error", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (c *ZaloChannel) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify webhook secret if configured
	if c.config.WebhookSecret != "" {
		secretHeader := r.Header.Get("X-Bot-Api-Secret-Token")
		if secretHeader != c.config.WebhookSecret {
			logger.WarnC("zalo", "Invalid webhook secret")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var update map[string]interface{}
	if err := json.Unmarshal(body, &update); err != nil {
		logger.ErrorCF("zalo", "Failed to parse webhook payload", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Return 200 immediately, process async
	w.WriteHeader(http.StatusOK)

	go c.processUpdate(update)
}

// --- Message Processing ---

func (c *ZaloChannel) processUpdate(update map[string]interface{}) {
	eventName, _ := update["event_name"].(string)

	switch eventName {
	case "message.text.received":
		c.processTextMessage(update)
	case "message.image.received":
		c.processImageMessage(update)
	case "message.sticker.received":
		c.processStickerMessage(update)
	default:
		logger.DebugCF("zalo", "Unhandled event", map[string]interface{}{
			"event": eventName,
		})
	}
}

func (c *ZaloChannel) processTextMessage(update map[string]interface{}) {
	message, ok := update["message"].(map[string]interface{})
	if !ok {
		return
	}

	text, _ := message["text"].(string)
	if text == "" {
		return
	}

	// Sender is at message.from.id
	from, _ := message["from"].(map[string]interface{})
	senderID, _ := from["id"].(string)
	if senderID == "" {
		return
	}

	// Chat ID is at message.chat.id
	chat, _ := message["chat"].(map[string]interface{})
	chatID, _ := chat["id"].(string)
	if chatID == "" {
		chatID = senderID
	}

	// Allow-list check
	if !c.IsAllowed(senderID) {
		logger.DebugCF("zalo", "Message from non-allowed user", map[string]interface{}{
			"sender_id": senderID,
		})
		return
	}

	logger.InfoCF("zalo", "Received text message", map[string]interface{}{
		"sender_id": senderID,
		"chat_id":   chatID,
		"text":      text,
	})

	c.HandleMessage(senderID, chatID, text, nil, nil)
}

func (c *ZaloChannel) processImageMessage(update map[string]interface{}) {
	// TODO: Implement image handling if needed
	logger.DebugC("zalo", "Image message received (not implemented)")
}

func (c *ZaloChannel) processStickerMessage(update map[string]interface{}) {
	// TODO: Implement sticker handling if needed
	logger.DebugC("zalo", "Sticker message received (not implemented)")
}

// --- Helpers ---

// chunkText splits text into chunks at newlines preferentially.
func chunkText(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}

		// Try to find a newline before maxLen
		chunk := text[:maxLen]
		if idx := strings.LastIndex(chunk, "\n"); idx > maxLen/2 {
			chunk = text[:idx]
			text = text[idx+1:]
		} else {
			text = text[maxLen:]
		}

		chunks = append(chunks, chunk)
	}

	return chunks
}
