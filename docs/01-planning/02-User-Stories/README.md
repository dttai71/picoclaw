---
spec_id: SPEC-0004
spec_name: "Zalo Channel User Stories"
spec_version: "1.0.0"
status: draft
tier: PROFESSIONAL
stage: "01"
category: functional
owner: "PM/PJM"
created: 2026-02-15
last_updated: 2026-02-15
related_adrs: ["ADR-002"]
related_specs: ["SPEC-0003"]
tags: ["zalo", "user-stories", "channel"]
priority: P0
effort: L
---

# User Stories: Zalo Channel

## US-ZALO-001: Chat via Zalo

**As a** Vietnamese user,
**I want to** chat with PicoClaw through my Zalo account,
**so that** I can use my preferred messaging platform without switching apps.

### Acceptance Criteria

- GIVEN a configured and running PicoClaw gateway with Zalo enabled
- WHEN I send a text message to the Zalo Official Account
- THEN PicoClaw receives the message via webhook
- AND processes it through the agent loop
- AND sends a response back to my Zalo chat
- AND the response appears within 5 seconds

---

## US-ZALO-002: Configure Zalo Credentials

**As an** administrator,
**I want to** configure Zalo OA credentials in `config.json`,
**so that** the bot connects to my Official Account.

### Acceptance Criteria

- GIVEN I have a Zalo OA with `app_id`, `app_secret`, and `oa_secret_key`
- WHEN I add them to `~/.picoclaw/config.json` under `channels.zalo`
- AND set `enabled: true`
- THEN `picoclaw gateway` starts the Zalo webhook server
- AND logs "Zalo channel enabled successfully"

---

## US-ZALO-003: OAuth Setup

**As an** administrator,
**I want to** run `picoclaw auth login --provider zalo` to complete OAuth authorization,
**so that** PicoClaw can send messages on behalf of my Official Account.

### Acceptance Criteria

- GIVEN Zalo `app_id` and `app_secret` are configured
- WHEN I run `picoclaw auth login --provider zalo`
- THEN the CLI prints an authorization URL
- AND opens my browser to the Zalo OAuth consent page
- AND after I authorize, tokens are saved to `~/.picoclaw/zalo_tokens.json`
- AND the file has 0600 permissions
- AND the CLI prints "OAuth successful"

---

## US-ZALO-004: Automatic Token Refresh

**As a** user,
**I want** OAuth tokens to refresh automatically,
**so that** I don't need to re-authorize frequently.

### Acceptance Criteria

- GIVEN valid tokens stored in `zalo_tokens.json`
- AND the access token is within 30 minutes of expiry
- WHEN the background refresh goroutine runs
- THEN a new access token is obtained via refresh token
- AND the new tokens are saved with 0600 permissions
- AND message sending continues without interruption

### Edge Cases

- IF refresh token has expired (>90 days): log warning, require user to run `picoclaw auth login --provider zalo` again
- IF refresh fails 3 times: stop retrying, log error with structured context

---

**Last Updated**: 2026-02-15
**Owner**: PM/PJM
**Related**: [Requirements](../01-Requirements/README.md), [ADR-002](../../02-design/04-ADRs/ADR-002-zalo-channel.md)
