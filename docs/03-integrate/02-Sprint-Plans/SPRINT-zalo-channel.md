---
spec_id: SPRINT-2026-003
spec_name: "Zalo Channel Integration Sprint"
spec_version: "1.0.0"
status: completed
tier: PROFESSIONAL
stage: "03"
category: integration
owner: "PM/PJM"
created: 2026-02-15
last_updated: 2026-02-15
completed: 2026-02-15
related_adrs: ["ADR-002"]
related_specs: ["SPEC-0003", "SPEC-0004"]
tags: ["sprint", "zalo", "channel", "integration"]
priority: P0
effort: L
---

# Sprint Plan: Zalo Channel Integration

**Sprint ID**: SPRINT-2026-003
**Duration**: 5 working days
**Goal**: Implement Zalo OA channel integration following LINE pattern
**Status**: Completed
**ADR**: [ADR-002](../../02-design/04-ADRs/ADR-002-zalo-channel.md)

---

## Prerequisites

- CTO approval of ADR-002
- Zalo OA test account with `app_id`, `app_secret`, `oa_secret_key`
- Zalo OA dashboard configured with webhook URL
- Public webhook URL for development (ngrok or similar)

---

## Sprint Backlog

### Day 1: Config + Types (4 hours)

- [x] Add `ZaloConfig` struct to `pkg/config/config.go` (~line 151)
- [x] Add `Zalo ZaloConfig` field to `ChannelsConfig` struct (~line 81)
- [x] Add Zalo defaults in `DefaultConfig()` (~line 297)
- [x] Verify env var overrides work: `PICOCLAW_CHANNELS_ZALO_*`
- [x] Run `go vet ./pkg/config/...` — pass

### Day 2: Core Channel Implementation (8 hours)

- [x] Create `pkg/channels/zalo.go`
  - [x] `ZaloChannel` struct (embed `*BaseChannel`)
  - [x] `NewZaloChannel()` constructor — validate AppID + OASecretKey
  - [x] `Start()` — load tokens, launch webhook HTTP server, rate limiter
  - [x] `Stop()` — graceful shutdown (cancel context, close HTTP server)
  - [x] `webhookHandler()` — POST check, read body, verify signature, parse event
  - [x] `verifySignature()` — HMAC-SHA256 with `oa_secret_key`, constant-time compare
  - [x] `processEvent()` — handle `user_send_text`, extract sender ID + content
  - [x] `Send()` — POST to Zalo OA API with access token + rate limiter
  - [x] `callAPI()` — authenticated POST helper (30s timeout, 401/429 handling)
- [x] Run `go vet ./pkg/channels/...` — pass

### Day 3: OAuth + Token Management (6 hours)

- [x] OAuth 2.0 authorization URL builder (`ZaloOAuthLogin()`)
- [x] OAuth callback handler for code exchange
- [x] `ExchangeCodeForToken()` — POST to `oauth.zaloapp.com/v4/oa/access_token`
- [x] `refreshToken()` — POST with `grant_type=refresh_token`
- [x] `refreshTokenLoop()` — background goroutine, check every 5min, refresh at <30min
- [x] `loadTokens()` — read from `~/.picoclaw/zalo_tokens.json`
- [x] `saveTokens()` — atomic write (temp + rename) with 0600 permissions
- [x] Token lifecycle: single pair only, ~1KB max file size (FR-ZALO-007)

### Day 4: Registration + CLI (4 hours)

- [x] Register Zalo in `pkg/channels/manager.go` `initChannels()` — after LINE block
- [x] Guard condition: `Enabled && AppID != ""`
- [x] Add `"zalo"` case in `authLoginCmd()` in `cmd/picoclaw/main.go`
- [x] Implement `authLoginZalo()` — start temp server, build URL, open browser, exchange code
- [x] Run `make build` — verify binary compiles

### Day 5: Tests + Documentation (6 hours)

- [x] Create `pkg/channels/zalo_test.go`:
  - [x] `TestNewZaloChannel` — valid config, missing app_id, missing oa_secret_key
  - [x] `TestZaloVerifySignature` — valid, invalid, empty, tampered body
  - [x] `TestZaloWebhookHandler` — POST/GET, valid/invalid signature, malformed JSON
  - [x] `TestZaloChannelIsAllowed` — empty allowlist, restricted users
  - [x] `TestZaloTokenManagement` — load/save, 0600 permissions, expiry detection
- [x] Create `docs/03-integrate/03-Setup-Guides/zalo-channel.md`
- [x] Update `docs/03-integrate/03-Setup-Guides/README.md` with Zalo entry
- [x] Run full test suite: `go test ./...`

---

## Integration Test Plan (CTO P1)

### 1. OAuth Flow Test

- Mock Zalo auth endpoint
- Verify code-to-token exchange
- Verify token refresh logic
- Verify token file persistence (0600)

### 2. Webhook Test

- Send test events with valid/invalid HMAC
- Verify signature rejection (HTTP 401)
- Verify event processing pipeline

### 3. Send Message Test

- Mock Zalo API endpoint
- Verify token auto-refresh triggers
- Verify message delivery
- Verify error handling (401, 429, timeout)

### 4. Concurrent Connections

- 5 simultaneous Zalo users sending messages
- Memory usage delta <5MB
- No race conditions (run with `-race` flag)

---

## Definition of Done

- [x] All tests pass: `go test ./pkg/channels/... -v -race`
- [x] `go vet ./...` passes
- [x] `make fmt` produces no diff
- [x] Cross-compile passes: `make build-all`
- [ ] Manual test: send message via Zalo OA, receive response (requires Zalo OA account)
- [x] ADR-002 status updated to "Accepted"
- [x] Setup guide complete and reviewed

---

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Zalo API rate limits undocumented | Medium | Medium | Start conservative, respect 429 + Retry-After, added ~100 msg/min rate limiter |
| OAuth token expiry during dev | Low | High | Mock token server in unit tests |
| Zalo API v3.0 undocumented behavior | Medium | Medium | Start with text-only, extend incrementally |
| Webhook port conflict in production | Low | Low | Configurable port, document reverse proxy setup |

---

## Dependencies

| Dependency | Owner | Status |
|------------|-------|--------|
| ADR-002 approval | CTO | Accepted |
| Zalo OA test account | PM | Pending |
| Public webhook URL (ngrok) | Dev | Available |

---

**Created**: 2026-02-15
**Completed**: 2026-02-15
**Owner**: PM/PJM
**Assigned**: Development Team
