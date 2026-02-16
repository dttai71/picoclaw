package channels

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	zcaRestartDelay    = 5 * time.Second
	zcaMaxRestartDelay = 5 * time.Minute
)

// zcaMessage represents a message from zca listen output.
type zcaMessage struct {
	ThreadID  string      `json:"threadId"`
	MsgID     string      `json:"msgId"`
	Type      int         `json:"type"`      // 1=text, others=media
	Content   string      `json:"content"`   // text content
	Timestamp int64       `json:"timestamp"` // Unix ms
	Metadata  zcaMetadata `json:"metadata"`
}

type zcaMetadata struct {
	ThreadType int    `json:"threadType"` // 1=DM, 2=group
	SenderID   string `json:"senderId"`
}

// ZaloUserChannel implements the Channel interface for Zalo Personal
// via the zca-cli binary (QR code authentication, CLI streaming).
type ZaloUserChannel struct {
	*BaseChannel
	config  config.ZaloUserConfig
	zcaPath string
	cmd     *exec.Cmd
	cmdMu   sync.Mutex
	sendMu  sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewZaloUserChannel creates a new Zalo Personal channel instance.
func NewZaloUserChannel(cfg config.ZaloUserConfig, messageBus *bus.MessageBus) (*ZaloUserChannel, error) {
	zcaPath := cfg.ZcaPath
	if zcaPath == "" {
		zcaPath = "zca"
	}

	// Verify zca binary exists
	path, err := exec.LookPath(zcaPath)
	if err != nil {
		return nil, fmt.Errorf("zca binary not found in PATH: %w (install from https://github.com/zalo-api/zca-cli)", err)
	}

	base := NewBaseChannel("zalouser", cfg, messageBus, cfg.AllowFrom)

	return &ZaloUserChannel{
		BaseChannel: base,
		config:      cfg,
		zcaPath:     path,
	}, nil
}

// Start launches the zca listen loop.
func (c *ZaloUserChannel) Start(ctx context.Context) error {
	logger.InfoC("zalouser", "Starting Zalo Personal channel (zca-cli)")

	c.ctx, c.cancel = context.WithCancel(ctx)

	go c.listenLoop()

	c.setRunning(true)
	logger.InfoC("zalouser", "Zalo Personal channel started")
	return nil
}

// Stop terminates the zca listener process.
func (c *ZaloUserChannel) Stop(ctx context.Context) error {
	logger.InfoC("zalouser", "Stopping Zalo Personal channel")

	if c.cancel != nil {
		c.cancel()
	}

	c.cmdMu.Lock()
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	c.cmdMu.Unlock()

	c.setRunning(false)
	logger.InfoC("zalouser", "Zalo Personal channel stopped")
	return nil
}

// Send sends a message via zca msg send.
func (c *ZaloUserChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("zalouser channel not running")
	}

	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	args := []string{"msg", "send", msg.ChatID, msg.Content}
	if c.config.Profile != "" {
		args = append([]string{"-p", c.config.Profile}, args...)
	}

	cmd := exec.CommandContext(ctx, c.zcaPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zca send failed: %w (output: %s)", err, string(output))
	}

	logger.DebugCF("zalouser", "Sent message", map[string]interface{}{
		"chat_id": msg.ChatID,
		"text":    msg.Content,
	})

	return nil
}

// --- Listener Loop ---

func (c *ZaloUserChannel) listenLoop() {
	backoff := zcaRestartDelay
	maxBackoff := zcaMaxRestartDelay

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		logger.InfoC("zalouser", "Starting zca listener")

		if err := c.runListener(); err != nil {
			logger.ErrorCF("zalouser", "zca listener error", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Auto-restart with exponential backoff
		logger.InfoCF("zalouser", "Restarting zca listener", map[string]interface{}{
			"delay": backoff.String(),
		})

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *ZaloUserChannel) runListener() error {
	args := []string{"listen", "-r", "-k"}
	if c.config.Profile != "" {
		args = append([]string{"-p", c.config.Profile}, args...)
	}

	c.cmdMu.Lock()
	c.cmd = exec.CommandContext(c.ctx, c.zcaPath, args...)
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		c.cmdMu.Unlock()
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		c.cmdMu.Unlock()
		return fmt.Errorf("start zca listen: %w", err)
	}
	c.cmdMu.Unlock()

	logger.InfoC("zalouser", "zca listen started")

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg zcaMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			logger.DebugCF("zalouser", "Failed to parse zca message", map[string]interface{}{
				"error": err.Error(),
				"line":  line,
			})
			continue
		}

		c.processMessage(msg)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	c.cmdMu.Lock()
	err = c.cmd.Wait()
	c.cmdMu.Unlock()

	if err != nil && c.ctx.Err() == nil {
		return fmt.Errorf("zca process exited: %w", err)
	}

	return nil
}

// --- Message Processing ---

func (c *ZaloUserChannel) processMessage(msg zcaMessage) {
	// Only handle text messages
	if msg.Type != 1 {
		return
	}

	if msg.Content == "" || msg.ThreadID == "" {
		return
	}

	senderID := msg.Metadata.SenderID
	if senderID == "" {
		senderID = msg.ThreadID // fallback for DMs
	}

	// Allow-list check
	if !c.IsAllowed(senderID) {
		logger.DebugCF("zalouser", "Message from non-allowed user", map[string]interface{}{
			"sender_id": senderID,
		})
		return
	}

	logger.DebugCF("zalouser", "Received text message", map[string]interface{}{
		"thread_id": msg.ThreadID,
		"sender_id": senderID,
		"type":      msg.Metadata.ThreadType,
		"text":      msg.Content,
	})

	c.HandleMessage(senderID, msg.ThreadID, msg.Content, nil, nil)
}
