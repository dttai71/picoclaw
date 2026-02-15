---
spec_id: SPRINT-2026-003
spec_name: "Zalo Channel Integration Sprint"
spec_version: "1.0.0"
status: draft
tier: PROFESSIONAL
stage: "03"
category: integration
owner: "PM/PJM"
created: 2026-02-15
last_updated: 2026-02-15
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
**Status**: Awaiting CTO Approval
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

- [ ] Add `ZaloConfig` struct to `pkg/config/config.go` (~line 149)
- [ ] Add `Zalo ZaloConfig` field to `ChannelsConfig` struct (~line 82)
- [ ] Add Zalo defaults in `DefaultConfig()` (~line 296)
- [ ] Verify env var overrides work: `PICOCLAW_CHANNELS_ZALO_*`
- [ ] Run `go vet ./pkg/config/...` — pass

### Day 2: Core Channel Implementation (8 hours)

- [ ] Create `pkg/channels/zalo.go`
  - [ ] `ZaloChannel` struct (embed `*BaseChannel`)
  - [ ] `NewZaloChannel()` constructor — validate AppID + OASecretKey
  - [ ] `Start()` — load tokens, launch webhook HTTP server
  - [ ] `Stop()` — graceful shutdown (cancel context, close HTTP server)
  - [ ] `webhookHandler()` — POST check, read body, verify signature, parse event
  - [ ] `verifySignature()` — HMAC-SHA256 with `oa_secret_key`, constant-time compare
  - [ ] `processEvent()` — handle `user_send_text`, extract sender ID + content
  - [ ] `Send()` — POST to Zalo OA API with access token
  - [ ] `callAPI()` — authenticated POST helper (30s timeout)
- [ ] Run `go vet ./pkg/channels/...` — pass

### Day 3: OAuth + Token Management (6 hours)

- [ ] OAuth 2.0 authorization URL builder (`InitiateOAuth()`)
- [ ] OAuth callback handler for code exchange
- [ ] `exchangeCodeForToken()` — POST to `oauth.zaloapp.com/v4/oa/access_token`
- [ ] `refreshToken()` — POST with `grant_type=refresh_token`
- [ ] `refreshTokenLoop()` — background goroutine, check every 5min, refresh at <30min
- [ ] `loadTokens()` — read from `~/.picoclaw/zalo_tokens.json`
- [ ] `saveTokens()` — atomic write (temp + rename) with 0600 permissions
- [ ] Token lifecycle: single pair only, ~1KB max file size (FR-ZALO-007)

### Day 4: Registration + CLI (4 hours)

- [ ] Register Zalo in `pkg/channels/manager.go` `initChannels()` — after LINE block
- [ ] Guard condition: `Enabled && AppID != ""`
- [ ] Add `"zalo"` case in `authLoginCmd()` in `cmd/picoclaw/main.go`
- [ ] Implement `authLoginZalo()` — start temp server, build URL, open browser, exchange code
- [ ] Update `printHelp()` with Zalo mention
- [ ] Run `make build` — verify binary compiles

### Day 5: Tests + Documentation (6 hours)

- [ ] Create `pkg/channels/zalo_test.go`:
  - [ ] `TestNewZaloChannel` — valid config, missing app_id, missing oa_secret_key
  - [ ] `TestZaloVerifySignature` — valid, invalid, empty, tampered body
  - [ ] `TestZaloWebhookHandler` — POST/GET, valid/invalid signature, malformed JSON
  - [ ] `TestZaloChannelIsAllowed` — empty allowlist, restricted users
  - [ ] `TestZaloTokenManagement` — load/save, 0600 permissions, expiry detection
- [ ] Create `docs/03-integrate/03-Setup-Guides/zalo-channel.md`
- [ ] Update `docs/03-integrate/03-Setup-Guides/README.md` with Zalo entry
- [ ] Run full test suite: `go test ./...`

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

- [ ] All tests pass: `go test ./pkg/channels/... -v -race`
- [ ] `go vet ./...` passes
- [ ] `make fmt` produces no diff
- [ ] Binary size remains <10MB: `ls -la build/picoclaw`
- [ ] Cross-compile passes: `make build-all`
- [ ] Manual test: send message via Zalo OA, receive response
- [ ] ADR-002 status updated to "Accepted"
- [ ] Setup guide complete and reviewed

---

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Zalo API rate limits undocumented | Medium | Medium | Start conservative, respect 429 + Retry-After |
| OAuth token expiry during dev | Low | High | Mock token server in unit tests |
| Zalo API v3.0 undocumented behavior | Medium | Medium | Start with text-only, extend incrementally |
| Webhook port conflict in production | Low | Low | Configurable port, document reverse proxy setup |

---

## Dependencies

| Dependency | Owner | Status |
|------------|-------|--------|
| ADR-002 approval | CTO | Pending |
| Zalo OA test account | PM | Pending |
| Public webhook URL (ngrok) | Dev | Available |

---

**Created**: 2026-02-15
**Owner**: PM/PJM
**Assigned**: Development Team
