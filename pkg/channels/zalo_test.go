package channels

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewZaloChannel(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ZaloConfig
		wantErr string
	}{
		{
			name: "valid config",
			cfg: config.ZaloConfig{
				Enabled:     true,
				AppID:       "test_app_id",
				AppSecret:   "test_secret",
				OASecretKey: "test_oa_key",
				WebhookHost: "0.0.0.0",
				WebhookPort: 18792,
				WebhookPath: "/webhook/zalo",
			},
			wantErr: "",
		},
		{
			name: "missing app_id",
			cfg: config.ZaloConfig{
				Enabled:     true,
				OASecretKey: "test_oa_key",
			},
			wantErr: "zalo app_id and oa_secret_key are required",
		},
		{
			name: "missing oa_secret_key",
			cfg: config.ZaloConfig{
				Enabled: true,
				AppID:   "test_app_id",
			},
			wantErr: "zalo app_id and oa_secret_key are required",
		},
		{
			name:    "both missing",
			cfg:     config.ZaloConfig{Enabled: true},
			wantErr: "zalo app_id and oa_secret_key are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, err := NewZaloChannel(tt.cfg, nil)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ch.Name() != "zalo" {
				t.Fatalf("expected channel name 'zalo', got %q", ch.Name())
			}
		})
	}
}

func TestZaloVerifySignature(t *testing.T) {
	secretKey := "test_oa_secret_key"
	ch := &ZaloChannel{
		config: config.ZaloConfig{OASecretKey: secretKey},
	}

	body := []byte(`{"event_name":"user_send_text","sender":{"id":"123"}}`)

	// Compute valid HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(body)
	validSignature := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		body      []byte
		signature string
		want      bool
	}{
		{
			name:      "valid signature",
			body:      body,
			signature: validSignature,
			want:      true,
		},
		{
			name:      "invalid signature",
			body:      body,
			signature: "deadbeef1234567890abcdef",
			want:      false,
		},
		{
			name:      "empty signature",
			body:      body,
			signature: "",
			want:      false,
		},
		{
			name:      "tampered body",
			body:      []byte(`{"event_name":"user_send_text","sender":{"id":"999"}}`),
			signature: validSignature,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ch.verifySignature(tt.body, tt.signature)
			if got != tt.want {
				t.Fatalf("verifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZaloWebhookHandler(t *testing.T) {
	secretKey := "webhook_test_key"
	mb := bus.NewMessageBus()
	ch := &ZaloChannel{
		BaseChannel: NewBaseChannel("zalo", nil, mb, nil),
		config:      config.ZaloConfig{OASecretKey: secretKey},
	}

	sign := func(body []byte) string {
		mac := hmac.New(sha256.New, []byte(secretKey))
		mac.Write(body)
		return hex.EncodeToString(mac.Sum(nil))
	}

	t.Run("GET method rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook/zalo", nil)
		w := httptest.NewRecorder()
		ch.webhookHandler(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", w.Code)
		}
	})

	t.Run("missing signature returns 401", func(t *testing.T) {
		body := []byte(`{"event_name":"user_send_text"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook/zalo", strings.NewReader(string(body)))
		w := httptest.NewRecorder()
		ch.webhookHandler(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid signature returns 401", func(t *testing.T) {
		body := []byte(`{"event_name":"user_send_text"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook/zalo", strings.NewReader(string(body)))
		req.Header.Set("X-ZaloOA-Signature", "invalid_signature")
		w := httptest.NewRecorder()
		ch.webhookHandler(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("valid signature returns 200", func(t *testing.T) {
		body := []byte(`{"event_name":"user_send_text","sender":{"id":"123"},"recipient":{"id":"456"},"message":{"msg_id":"m1","text":"hello"}}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook/zalo", strings.NewReader(string(body)))
		req.Header.Set("X-ZaloOA-Signature", sign(body))
		w := httptest.NewRecorder()
		ch.webhookHandler(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		body := []byte(`{invalid json}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook/zalo", strings.NewReader(string(body)))
		req.Header.Set("X-ZaloOA-Signature", sign(body))
		w := httptest.NewRecorder()
		ch.webhookHandler(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestZaloChannelIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowFrom []string
		senderID  string
		want      bool
	}{
		{
			name:      "empty allowlist allows all",
			allowFrom: nil,
			senderID:  "user123",
			want:      true,
		},
		{
			name:      "allowed user passes",
			allowFrom: []string{"user123"},
			senderID:  "user123",
			want:      true,
		},
		{
			name:      "denied user blocked",
			allowFrom: []string{"user123"},
			senderID:  "user456",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ZaloConfig{
				Enabled:     true,
				AppID:       "test",
				OASecretKey: "test",
				AllowFrom:   config.FlexibleStringSlice(tt.allowFrom),
			}
			ch, err := NewZaloChannel(cfg, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := ch.IsAllowed(tt.senderID); got != tt.want {
				t.Fatalf("IsAllowed(%q) = %v, want %v", tt.senderID, got, tt.want)
			}
		})
	}
}

func TestZaloTokenManagement(t *testing.T) {
	// Use temp directory for token file
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create .picoclaw directory
	picoDir := filepath.Join(tmpDir, ".picoclaw")
	if err := os.MkdirAll(picoDir, 0755); err != nil {
		t.Fatalf("failed to create .picoclaw dir: %v", err)
	}

	ch := &ZaloChannel{
		config: config.ZaloConfig{
			AppID:     "test_app",
			AppSecret: "test_secret",
		},
	}

	t.Run("load missing file returns error", func(t *testing.T) {
		err := ch.loadTokens()
		if err == nil {
			t.Fatal("expected error loading missing tokens, got nil")
		}
	})

	t.Run("save and load tokens", func(t *testing.T) {
		tokens := &zaloTokens{
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		if err := ch.saveTokens(tokens); err != nil {
			t.Fatalf("failed to save tokens: %v", err)
		}

		// Verify file permissions
		path := filepath.Join(picoDir, zaloTokenFileName)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat token file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
		}

		// Verify file content is valid JSON
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read token file: %v", err)
		}
		var loaded zaloTokens
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("token file is not valid JSON: %v", err)
		}
		if loaded.AccessToken != "test_access_token" {
			t.Fatalf("expected access_token 'test_access_token', got %q", loaded.AccessToken)
		}
		if loaded.RefreshToken != "test_refresh_token" {
			t.Fatalf("expected refresh_token 'test_refresh_token', got %q", loaded.RefreshToken)
		}

		// Load tokens via channel method
		ch2 := &ZaloChannel{config: ch.config}
		if err := ch2.loadTokens(); err != nil {
			t.Fatalf("failed to load tokens: %v", err)
		}
		ch2.tokenMu.RLock()
		if ch2.tokens.AccessToken != "test_access_token" {
			t.Fatalf("loaded access_token mismatch: %q", ch2.tokens.AccessToken)
		}
		ch2.tokenMu.RUnlock()
	})

	t.Run("token expiry detection", func(t *testing.T) {
		// Token that expires in 10 minutes (within refresh buffer)
		expiringSoon := &zaloTokens{
			AccessToken:  "expiring_soon",
			RefreshToken: "refresh",
			ExpiresAt:    time.Now().Add(10 * time.Minute),
		}
		if time.Until(expiringSoon.ExpiresAt) > zaloRefreshBuffer {
			t.Fatal("expected token to be within refresh buffer")
		}

		// Token that expires in 2 hours (outside refresh buffer)
		expiringLater := &zaloTokens{
			AccessToken:  "expiring_later",
			RefreshToken: "refresh",
			ExpiresAt:    time.Now().Add(2 * time.Hour),
		}
		if time.Until(expiringLater.ExpiresAt) <= zaloRefreshBuffer {
			t.Fatal("expected token to be outside refresh buffer")
		}
	})
}
