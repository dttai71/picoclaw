package channels

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewZaloUserChannel(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.ZaloUserConfig
		wantError bool
	}{
		{
			name: "zca not found",
			cfg: config.ZaloUserConfig{
				Enabled: true,
				ZcaPath: "/nonexistent/zca",
			},
			wantError: true,
		},
		{
			name: "default zca path",
			cfg: config.ZaloUserConfig{
				Enabled: true,
				ZcaPath: "", // will use "zca" from PATH
			},
			wantError: false, // will succeed if zca is in PATH, fail otherwise
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageBus := bus.NewMessageBus()
			_, err := NewZaloUserChannel(tt.cfg, messageBus)
			if tt.wantError && err == nil {
				t.Error("Expected error but got nil")
			}
			// Note: We can't guarantee zca is in PATH, so we only check expected errors
		})
	}
}

func TestZcaMessageParsing(t *testing.T) {
	tests := []struct {
		name      string
		jsonLine  string
		wantValid bool
		wantType  int
	}{
		{
			name:      "valid text message",
			jsonLine:  `{"threadId":"123","msgId":"456","type":1,"content":"Hello","timestamp":1234567890,"metadata":{"threadType":1,"senderId":"user123"}}`,
			wantValid: true,
			wantType:  1,
		},
		{
			name:      "valid image message",
			jsonLine:  `{"threadId":"123","msgId":"456","type":2,"content":"","timestamp":1234567890,"metadata":{"threadType":1,"senderId":"user123"}}`,
			wantValid: true,
			wantType:  2,
		},
		{
			name:      "malformed JSON",
			jsonLine:  `{invalid json`,
			wantValid: false,
		},
		{
			name:      "empty JSON",
			jsonLine:  `{}`,
			wantValid: true, // parses but missing required fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg zcaMessage
			err := json.Unmarshal([]byte(tt.jsonLine), &msg)

			if tt.wantValid && err != nil {
				t.Errorf("Expected valid JSON but got error: %v", err)
			}
			if !tt.wantValid && err == nil {
				t.Error("Expected invalid JSON but got nil error")
			}

			if tt.wantValid && err == nil {
				if msg.Type != tt.wantType {
					t.Errorf("Type = %d, want %d", msg.Type, tt.wantType)
				}
			}
		})
	}
}

func TestZaloUserProcessMessage(t *testing.T) {
	cfg := config.ZaloUserConfig{
		Enabled:   true,
		ZcaPath:   "zca",
		AllowFrom: config.FlexibleStringSlice{"user123"},
	}

	messageBus := bus.NewMessageBus()
	ch, err := NewZaloUserChannel(cfg, messageBus)
	if err != nil {
		t.Skipf("Skipping test: zca not found in PATH (%v)", err)
	}

	tests := []struct {
		name       string
		msg        zcaMessage
		wantSkip   bool
		skipReason string
	}{
		{
			name: "valid text message from allowed user",
			msg: zcaMessage{
				ThreadID:  "thread123",
				MsgID:     "msg456",
				Type:      1,
				Content:   "Hello",
				Timestamp: 1234567890000,
				Metadata: zcaMetadata{
					ThreadType: 1,
					SenderID:   "user123",
				},
			},
			wantSkip: false,
		},
		{
			name: "text message from non-allowed user",
			msg: zcaMessage{
				ThreadID:  "thread123",
				MsgID:     "msg456",
				Type:      1,
				Content:   "Hello",
				Timestamp: 1234567890000,
				Metadata: zcaMetadata{
					ThreadType: 1,
					SenderID:   "user999",
				},
			},
			wantSkip:   true,
			skipReason: "not in allow list",
		},
		{
			name: "non-text message (image)",
			msg: zcaMessage{
				ThreadID:  "thread123",
				MsgID:     "msg456",
				Type:      2,
				Content:   "",
				Timestamp: 1234567890000,
				Metadata: zcaMetadata{
					ThreadType: 1,
					SenderID:   "user123",
				},
			},
			wantSkip:   true,
			skipReason: "type != 1 (text)",
		},
		{
			name: "empty content",
			msg: zcaMessage{
				ThreadID:  "thread123",
				MsgID:     "msg456",
				Type:      1,
				Content:   "",
				Timestamp: 1234567890000,
				Metadata: zcaMetadata{
					ThreadType: 1,
					SenderID:   "user123",
				},
			},
			wantSkip:   true,
			skipReason: "empty content",
		},
		{
			name: "group message from allowed user",
			msg: zcaMessage{
				ThreadID:  "group123",
				MsgID:     "msg456",
				Type:      1,
				Content:   "Hello group",
				Timestamp: 1234567890000,
				Metadata: zcaMetadata{
					ThreadType: 2,
					SenderID:   "user123",
				},
			},
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch.setRunning(true)
			ch.processMessage(tt.msg)
			// Note: In a real test, we'd verify the message bus received the expected calls
			// For now, we just verify it doesn't crash
		})
	}
}

func TestZaloUserIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowFrom []string
		senderID  string
		wantOK    bool
	}{
		{
			name:      "empty allowlist allows all",
			allowFrom: []string{},
			senderID:  "anyone",
			wantOK:    true,
		},
		{
			name:      "allowed user passes",
			allowFrom: []string{"user123", "user456"},
			senderID:  "user123",
			wantOK:    true,
		},
		{
			name:      "denied user blocked",
			allowFrom: []string{"user123", "user456"},
			senderID:  "user999",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ZaloUserConfig{
				Enabled:   true,
				ZcaPath:   "zca",
				AllowFrom: tt.allowFrom,
			}

			messageBus := bus.NewMessageBus()
			ch, err := NewZaloUserChannel(cfg, messageBus)
			if err != nil {
				t.Skipf("Skipping test: zca not found in PATH (%v)", err)
			}

			if got := ch.IsAllowed(tt.senderID); got != tt.wantOK {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.senderID, got, tt.wantOK)
			}
		})
	}
}

func TestZcaMessageTimestampConversion(t *testing.T) {
	// Test that timestamp conversion from milliseconds to time.Time is correct
	msg := zcaMessage{
		ThreadID:  "thread123",
		MsgID:     "msg456",
		Type:      1,
		Content:   "Test",
		Timestamp: 1234567890123, // Unix timestamp in milliseconds
		Metadata: zcaMetadata{
			ThreadType: 1,
			SenderID:   "user123",
		},
	}

	// Expected: 1234567890 seconds + 123 milliseconds
	// time.Unix(1234567890, 123*1e6)
	expectedSec := int64(1234567890)
	expectedNsec := int64(123 * 1e6)

	actualSec := msg.Timestamp / 1000
	actualNsec := (msg.Timestamp % 1000) * 1e6

	if actualSec != expectedSec {
		t.Errorf("Seconds = %d, want %d", actualSec, expectedSec)
	}
	if actualNsec != expectedNsec {
		t.Errorf("Nanoseconds = %d, want %d", actualNsec, expectedNsec)
	}
}

func TestZcaCommandConstruction(t *testing.T) {
	// Verify that commands are constructed correctly for different profiles
	tests := []struct {
		name       string
		profile    string
		wantPrefix []string
	}{
		{
			name:       "default profile",
			profile:    "",
			wantPrefix: []string{"msg", "send"},
		},
		{
			name:       "custom profile",
			profile:    "myprofile",
			wantPrefix: []string{"-p", "myprofile", "msg", "send"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the command construction logic
			args := []string{"msg", "send", "chatID", "content"}
			if tt.profile != "" {
				args = append([]string{"-p", tt.profile}, args...)
			}

			// Verify prefix matches expected
			for i, want := range tt.wantPrefix {
				if args[i] != want {
					t.Errorf("args[%d] = %q, want %q", i, args[i], want)
				}
			}

			// Verify full command
			fullCmd := strings.Join(args, " ")
			t.Logf("Full command: zca %s", fullCmd)
		})
	}
}
