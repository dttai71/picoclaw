package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	waMaxReconnectAttempts = 10
	waMaxReconnectDelay    = 5 * time.Minute
)

type WhatsAppChannel struct {
	*BaseChannel
	conn      *websocket.Conn
	config    config.WhatsAppConfig
	url       string
	mu        sync.Mutex
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewWhatsAppChannel(cfg config.WhatsAppConfig, bus *bus.MessageBus) (*WhatsAppChannel, error) {
	base := NewBaseChannel("whatsapp", cfg, bus, cfg.AllowFrom)

	return &WhatsAppChannel{
		BaseChannel: base,
		config:      cfg,
		url:         cfg.BridgeURL,
		connected:   false,
	}, nil
}

func (c *WhatsAppChannel) Start(ctx context.Context) error {
	logger.InfoCF("whatsapp", "Starting WhatsApp channel", map[string]interface{}{
		"url": c.url,
	})

	c.ctx, c.cancel = context.WithCancel(ctx)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WhatsApp bridge: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	c.setRunning(true)
	logger.InfoC("whatsapp", "WhatsApp channel connected")

	go c.listen()

	return nil
}

func (c *WhatsAppChannel) Stop(ctx context.Context) error {
	logger.InfoC("whatsapp", "Stopping WhatsApp channel")

	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			logger.ErrorCF("whatsapp", "Error closing connection", map[string]interface{}{
				"error": err.Error(),
			})
		}
		c.conn = nil
	}

	c.connected = false
	c.setRunning(false)

	return nil
}

func (c *WhatsAppChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("whatsapp connection not established")
	}

	payload := map[string]interface{}{
		"type":    "message",
		"to":      msg.ChatID,
		"content": msg.Content,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (c *WhatsAppChannel) listen() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				logger.ErrorCF("whatsapp", "Read error", map[string]interface{}{
					"error": err.Error(),
				})

				c.mu.Lock()
				c.connected = false
				if c.conn != nil {
					c.conn.Close()
					c.conn = nil
				}
				c.mu.Unlock()

				// Attempt reconnect
				go c.reconnect()
				return
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				logger.WarnCF("whatsapp", "Failed to unmarshal message", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			msgType, ok := msg["type"].(string)
			if !ok {
				continue
			}

			switch msgType {
			case "message":
				c.handleIncomingMessage(msg)
			case "status":
				status, _ := msg["status"].(string)
				logger.InfoCF("whatsapp", "Bridge status update", map[string]interface{}{
					"status": status,
				})
			case "error":
				code, _ := msg["code"].(string)
				errMsg, _ := msg["message"].(string)
				logger.ErrorCF("whatsapp", "Bridge error", map[string]interface{}{
					"code":    code,
					"message": errMsg,
				})
			}
		}
	}
}

func (c *WhatsAppChannel) reconnect() {
	for attempt := 0; attempt < waMaxReconnectAttempts; attempt++ {
		delay := time.Duration(math.Min(
			float64(time.Second*(1<<uint(attempt))),
			float64(waMaxReconnectDelay),
		))

		logger.InfoCF("whatsapp", "Reconnecting", map[string]interface{}{
			"attempt": attempt + 1,
			"delay":   delay.String(),
		})

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(delay):
		}

		dialer := websocket.DefaultDialer
		dialer.HandshakeTimeout = 10 * time.Second

		conn, _, err := dialer.Dial(c.url, nil)
		if err != nil {
			logger.WarnCF("whatsapp", "Reconnect failed", map[string]interface{}{
				"attempt": attempt + 1,
				"error":   err.Error(),
			})
			continue
		}

		c.mu.Lock()
		c.conn = conn
		c.connected = true
		c.mu.Unlock()

		logger.InfoC("whatsapp", "Reconnected successfully")
		go c.listen()
		return
	}

	logger.ErrorC("whatsapp", "Max reconnect attempts reached, giving up")
}

func (c *WhatsAppChannel) handleIncomingMessage(msg map[string]interface{}) {
	senderID, ok := msg["from"].(string)
	if !ok {
		return
	}

	chatID, ok := msg["chat"].(string)
	if !ok {
		chatID = senderID
	}

	content, ok := msg["content"].(string)
	if !ok {
		content = ""
	}

	var mediaPaths []string
	if mediaData, ok := msg["media"].([]interface{}); ok {
		mediaPaths = make([]string, 0, len(mediaData))
		for _, m := range mediaData {
			if path, ok := m.(string); ok {
				mediaPaths = append(mediaPaths, path)
			}
		}
	}

	metadata := make(map[string]string)
	if messageID, ok := msg["id"].(string); ok {
		metadata["message_id"] = messageID
	}
	if userName, ok := msg["from_name"].(string); ok {
		metadata["user_name"] = userName
	}

	logger.DebugCF("whatsapp", "Received message", map[string]interface{}{
		"from":    senderID,
		"preview": utils.Truncate(content, 50),
	})

	c.HandleMessage(senderID, chatID, content, mediaPaths, metadata)
}
