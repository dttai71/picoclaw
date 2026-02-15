---
spec_id: SPEC-0003
spec_name: "Zalo Channel Integration Requirements"
spec_version: "1.0.0"
status: draft
tier: PROFESSIONAL
stage: "01"
category: functional
owner: "PM/PJM"
created: 2026-02-15
last_updated: 2026-02-15
related_adrs: ["ADR-002"]
related_specs: []
tags: ["zalo", "requirements", "channel"]
priority: P0
effort: L
---

# Requirements: Zalo Channel Integration

## Functional Requirements

### FR-ZALO-001: Webhook Message Receiving

Receive incoming messages from Zalo Official Account via webhook.

- System SHALL listen for POST requests on configurable webhook path (default: `/webhook/zalo`)
- System SHALL parse `user_send_text` events and extract sender ID and message content
- System SHALL return HTTP 200 immediately, processing events asynchronously
- System SHALL support configurable webhook host/port (default: `0.0.0.0:18792`)

### FR-ZALO-002: Message Sending

Send messages to Zalo users via OA API.

- System SHALL send text messages via `POST https://openapi.zalo.me/v3.0/oa/message/cs`
- System SHALL include valid access token in request header
- System SHALL handle API errors and log failures with structured context

### FR-ZALO-003: OAuth 2.0 Authorization

Authenticate with Zalo using OAuth 2.0 flow.

- System SHALL build authorization URL with `app_id` and `redirect_uri`
- System SHALL start temporary local HTTP server for OAuth callback
- System SHALL exchange authorization code for access token and refresh token
- System SHALL support CLI command `picoclaw auth login --provider zalo`

### FR-ZALO-004: Webhook Signature Verification

Verify webhook authenticity using HMAC-SHA256.

- System SHALL compute HMAC-SHA256 using `oa_secret_key`
- System SHALL verify against `X-ZaloOA-Signature` header
- System SHALL reject requests with invalid signatures (HTTP 401)
- System SHALL use constant-time comparison (`hmac.Equal`)

### FR-ZALO-005: Token Auto-Refresh

Automatically refresh expiring OAuth tokens.

- System SHALL check token expiry every 5 minutes via background goroutine
- System SHALL refresh tokens when less than 30 minutes remain before expiry
- System SHALL POST to `https://oauth.zaloapp.com/v4/oa/access_token` with `grant_type=refresh_token`
- System SHALL retry refresh 3 times with exponential backoff on failure
- System SHALL notify user to re-authenticate if refresh token expired (90 days)

### FR-ZALO-006: Secure Token Storage

Store OAuth tokens securely on local filesystem.

- System SHALL store tokens at `~/.picoclaw/zalo_tokens.json`
- System SHALL write token file with 0600 permissions (owner read/write only)
- System SHALL load stored tokens on channel startup
- System SHALL NOT log token values

### FR-ZALO-007: Token Lifecycle Management (CTO P0)

Manage token rotation and cleanup.

- Access token: 24h expiry, refresh at <30min remaining
- Refresh token: 90d expiry, user must re-authenticate if expired
- Token cleanup: Delete old access token after refresh, keep only current token pair
- File size: `zalo_tokens.json` max ~1KB (single token pair, no accumulation)
- Atomic writes: write to temp file, then rename (prevent corruption)

## Non-Functional Requirements

### NFR-ZALO-001: Zero New Dependencies

- System SHALL NOT add new Go module dependencies
- System SHALL use only standard library + existing go.mod packages
- Binary size increase SHALL be <100KB

### NFR-ZALO-002: Memory Efficiency

- Channel memory overhead SHALL be <500KB
- No unbounded data structures (maps, slices, channels)
- HTTP client with 30-second timeout (no hanging connections)

### NFR-ZALO-003: Cross-Platform Compatibility

- System SHALL compile for: linux/{amd64,arm64,riscv64}, darwin/arm64, windows/amd64
- System SHALL NOT use platform-specific APIs
- Token file paths SHALL use `filepath.Join` for OS-portable paths

### NFR-ZALO-004: Isolation

- Zalo channel code SHALL be contained in `pkg/channels/zalo.go`
- Zalo channel SHALL NOT share mutable state with other channels
- Disabling Zalo (`enabled: false`) SHALL have zero impact on other channels

---

**Last Updated**: 2026-02-15
**Owner**: PM/PJM
**Related**: [ADR-002](../../02-design/04-ADRs/ADR-002-zalo-channel.md)
