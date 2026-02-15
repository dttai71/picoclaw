---
spec_id: ADR-002
spec_name: "Zalo Channel Architecture"
spec_version: "1.0.0"
status: proposed
tier: PROFESSIONAL
stage: "02"
category: technical
owner: "CTO"
created: 2026-02-15
last_updated: 2026-02-15
related_adrs: ["ADR-001"]
related_specs: ["SPEC-0003"]
tags: ["zalo", "channel", "oauth", "webhook"]
priority: P0
effort: L
---

# ADR-002: Zalo Channel Architecture

**Date**: 2026-02-15
**Status**: Proposed
**Deciders**: CTO

## Context

PicoClaw needs a Zalo Official Account (OA) channel to serve the Vietnamese market (74M+ active users). Zalo is the dominant messaging platform in Vietnam. The solution must fit within the ultra-lightweight philosophy (<10MB RAM, <10MB binary).

Unlike LINE which uses permanent channel access tokens, Zalo OA uses OAuth 2.0 with short-lived access tokens (~24h) and refresh tokens (~90 days), requiring automatic token management.

## Decision Drivers

- Memory footprint: channel must add <500KB to the total application RAM
- Security: OAuth tokens stored locally with 0600 permissions, HMAC-SHA256 webhook verification
- Simplicity: follow the LINE channel pattern exactly, no new Go module dependencies
- Portability: must compile and run on RISC-V, ARM64, x86_64
- Vietnam market: Zalo is the primary messenger; LINE pattern is the closest architectural match
- Isolation: Zalo code must be fully isolated — disabling it has zero impact on other channels

### Zalo OA API Constraints

- API version: v3.0 (v2.0 deprecated)
- Send endpoint: `https://openapi.zalo.me/v3.0/oa/message/cs`
- OAuth endpoint: `https://oauth.zaloapp.com/v4/oa/access_token`
- Rate limits: research needed before implementation (typically 100-1000 msg/min for OA)
- Messaging model: 1:1 only (OA does not support group chats)
- Event types: `user_send_text`, `user_send_image`, `user_send_file`, `follow`, `unfollow`

## Considered Options

1. **Direct Zalo OA API integration following LINE pattern** (chosen)
2. Bridge/proxy pattern (like WhatsApp's WebSocket bridge)
3. Third-party Zalo SDK wrapper
4. Zalo Mini App approach

## Decision

Option 1: Direct HTTP integration with Zalo OA API v3.0.

### Architecture

ZaloChannel struct embeds `*BaseChannel` (same pattern as LINE, Telegram, etc.):

- HTTP webhook server receives events from Zalo
- REST API for sending messages via `https://openapi.zalo.me/v3.0/oa/message/cs`
- OAuth 2.0 flow via `https://oauth.zaloapp.com/v4/oa/`
- All code contained in single file: `pkg/channels/zalo.go`

### OAuth 2.0 Token Management

- Zalo uses short-lived access tokens (no permanent channel token like LINE)
- Access token: valid ~24 hours
- Refresh token: valid ~90 days
- Background goroutine checks expiry every 5 minutes, refreshes proactively at <30 min remaining
- Token storage: `~/.picoclaw/zalo_tokens.json` with `0600` permissions
- CLI flow: `picoclaw auth login --provider zalo` opens browser for OAuth consent
- Atomic writes: write to temp file then rename (prevent corruption on crash)
- Token cleanup: only current token pair stored, max ~1KB file size

**Rejected**: Storing tokens in `config.json` (tokens rotate frequently, config is for static settings).

### Webhook Verification

- Zalo sends signature in `X-ZaloOA-Signature` header
- Verify using HMAC-SHA256 with `oa_secret_key`
- Constant-time comparison via `hmac.Equal()` (same as LINE's signature verification)
- Invalid signatures rejected with HTTP 401

### Message Types (Phase 1)

- **Receive**: text only (`user_send_text` event)
- **Send**: text only (via OA message API)
- Image, file, sticker, location: deferred to Phase 2

### Configuration

```go
type ZaloConfig struct {
    Enabled     bool                `json:"enabled"`
    AppID       string              `json:"app_id"`
    AppSecret   string              `json:"app_secret"`
    OASecretKey string              `json:"oa_secret_key"`
    WebhookHost string              `json:"webhook_host"`
    WebhookPort int                 `json:"webhook_port"`
    WebhookPath string              `json:"webhook_path"`
    AllowFrom   FlexibleStringSlice `json:"allow_from"`
}
```

Port 18792 (avoids conflict with LINE/Web at 18791).

### Error Handling Strategy (CTO P2)

| Scenario | Action |
|----------|--------|
| OAuth code invalid | Return friendly error, log details |
| Token refresh fails | Retry 3x with exponential backoff, then require re-auth |
| API 401 (token revoked) | Clear stored tokens, log warning to re-authenticate |
| Network timeout | 30-second HTTP client timeout, structured error logging |
| Webhook signature invalid | Log + reject with HTTP 401 |
| Zalo API rate exceeded (429) | Log warning, drop message, respect `Retry-After` header |

## Consequences

### Positive

- Consistent with LINE pattern — easy to maintain, low learning curve
- No new Go module dependencies — binary size increase <100KB
- Opens Vietnam market (74M+ users)
- OAuth token management pattern reusable for future channels
- Full code isolation in single file

### Negative

- OAuth adds complexity vs simple token channels (LINE, Telegram)
- Background goroutine for token refresh (additional resource)
- Zalo OA is 1:1 only (no group chat support)
- External API dependency — subject to Zalo's uptime and API changes

### Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Zalo API breaking changes | Medium | Version lock to v3.0, monitor developer announcements |
| Token storage corruption | Medium | Atomic writes (temp + rename), validation on load |
| OAuth token expiry cascade | Low | Proactive refresh at 30-min buffer, 3x retry |
| Memory leak from HTTP clients | Low | 30-second timeout, defer resp.Body.Close() |

### Rollback Strategy (CTO P0)

1. **Feature flag**: `PICOCLAW_CHANNELS_ZALO_ENABLED=false` disables channel instantly
2. **Code isolation**: All Zalo code in `pkg/channels/zalo.go`, no shared mutable state
3. **Rollback procedure**:
   - Set `enabled: false` in config or env var
   - Restart gateway
   - Remove `~/.picoclaw/zalo_tokens.json` if corrupt
4. **Zero impact**: Other channels unaffected (tested in isolation)
5. **Binary rollback**: Previous binary version works without Zalo code

## Related

- [ADR-001: Web Channel Architecture](ADR-001-web-channel.md)
- [Requirements: FR-ZALO-001 through FR-ZALO-007](../../01-planning/01-Requirements/README.md)
- [User Stories: US-ZALO-001 through US-ZALO-004](../../01-planning/02-User-Stories/README.md)
