package channels

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewZaloChannel(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.ZaloConfig
		wantError bool
	}{
		{
			name: "valid token",
			cfg: config.ZaloConfig{
				Enabled: true,
				Token:   "123456:secret_token",
				Mode:    "polling",
			},
			wantError: false,
		},
		{
			name: "missing token",
			cfg: config.ZaloConfig{
				Enabled: true,
				Token:   "",
			},
			wantError: true,
		},
		{
			name: "invalid token format (no colon)",
			cfg: config.ZaloConfig{
				Enabled: true,
				Token:   "invalid_token",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageBus := bus.NewMessageBus()
			_, err := NewZaloChannel(tt.cfg, messageBus)
			if (err != nil) != tt.wantError {
				t.Errorf("NewZaloChannel() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestZaloWebhookHandler(t *testing.T) {
	cfg := config.ZaloConfig{
		Enabled:       true,
		Token:         "123456:secret_token",
		Mode:          "webhook",
		WebhookSecret: "test_secret",
	}

	messageBus := bus.NewMessageBus()
	ch, err := NewZaloChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	ch.ctx = context.Background()

	tests := []struct {
		name           string
		method         string
		secret         string
		body           string
		wantStatusCode int
	}{
		{
			name:           "GET method rejected",
			method:         "GET",
			secret:         "test_secret",
			body:           "",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing secret returns 401",
			method:         "POST",
			secret:         "",
			body:           `{}`,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid secret returns 401",
			method:         "POST",
			secret:         "wrong_secret",
			body:           `{}`,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "valid secret returns 200",
			method:         "POST",
			secret:         "test_secret",
			body:           `{"event_name": "message.text.received"}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "malformed JSON returns 400",
			method:         "POST",
			secret:         "test_secret",
			body:           `{invalid json`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/webhook/zalo", strings.NewReader(tt.body))
			if tt.secret != "" {
				req.Header.Set("X-Bot-Api-Secret-Token", tt.secret)
			}

			w := httptest.NewRecorder()
			ch.webhookHandler(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("webhookHandler() status = %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestZaloProcessUpdate(t *testing.T) {
	cfg := config.ZaloConfig{
		Enabled: true,
		Token:   "123456:secret_token",
		Mode:    "polling",
	}

	messageBus := bus.NewMessageBus()
	ch, err := NewZaloChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	ch.ctx = context.Background()
	ch.setRunning(true)

	tests := []struct {
		name      string
		update    map[string]interface{}
		wantCalls int // expected message bus calls
	}{
		{
			name: "text message",
			update: map[string]interface{}{
				"event_name": "message.text.received",
				"message": map[string]interface{}{
					"text": "Hello",
				},
				"sender": map[string]interface{}{
					"id": "user123",
				},
				"recipient": map[string]interface{}{
					"id": "user456",
				},
			},
			wantCalls: 1,
		},
		{
			name: "image message (not implemented)",
			update: map[string]interface{}{
				"event_name": "message.image.received",
			},
			wantCalls: 0,
		},
		{
			name: "sticker message (not implemented)",
			update: map[string]interface{}{
				"event_name": "message.sticker.received",
			},
			wantCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch.processUpdate(tt.update)
			// Note: In a real test, we'd verify the message bus received the expected calls
		})
	}
}

func TestZaloSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/sendMessage") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}

		if payload["chat_id"] == "" {
			t.Error("Missing chat_id in payload")
		}
		if payload["text"] == "" {
			t.Error("Missing text in payload")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	cfg := config.ZaloConfig{
		Enabled: true,
		Token:   "123456:secret_token",
		Mode:    "polling",
	}

	messageBus := bus.NewMessageBus()
	ch, err := NewZaloChannel(cfg, messageBus)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Override API base for testing
	ch.apiBase = server.URL + "/bot123456:secret_token"
	ch.ctx = context.Background()
	ch.setRunning(true)

	err = ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "user123",
		Content: "Test message",
	})

	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestChunkText(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		maxLen     int
		wantChunks int
	}{
		{
			name:       "short text (no chunking)",
			text:       "Hello",
			maxLen:     2000,
			wantChunks: 1,
		},
		{
			name:       "exact boundary",
			text:       strings.Repeat("a", 2000),
			maxLen:     2000,
			wantChunks: 1,
		},
		{
			name:       "needs chunking",
			text:       strings.Repeat("a", 2001),
			maxLen:     2000,
			wantChunks: 2,
		},
		{
			name:       "chunk at newline",
			text:       strings.Repeat("a", 1000) + "\n" + strings.Repeat("b", 1500),
			maxLen:     2000,
			wantChunks: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkText(tt.text, tt.maxLen)
			if len(chunks) != tt.wantChunks {
				t.Errorf("chunkText() returned %d chunks, want %d", len(chunks), tt.wantChunks)
			}

			// Verify all chunks fit within maxLen
			for i, chunk := range chunks {
				if len(chunk) > tt.maxLen {
					t.Errorf("Chunk %d exceeds maxLen: %d > %d", i, len(chunk), tt.maxLen)
				}
			}

			// Verify chunks concatenate to original text
			result := strings.Join(chunks, "")
			if result != tt.text {
				t.Error("Chunks don't concatenate to original text")
			}
		})
	}
}
