# ADR-001: Web Channel Architecture

**Date**: 2026-02-15
**Status**: Accepted
**Deciders**: CTO (3 review rounds)

## Context

PicoClaw needs a browser-based chat interface and dashboard. The solution must fit within the ultra-lightweight philosophy (<10MB RAM total, <10MB binary).

## Decision Drivers

- Memory footprint: entire application must stay under 10MB RAM
- Security: runs on personal machines with API keys and conversation history
- Simplicity: no external dependencies beyond gorilla/websocket (already in go.mod)
- Portability: must work on RISC-V, ARM64, x86_64

## Considered Options

1. **Embedded single-file HTML + Go HTTP/WS server** (chosen)
2. SPA framework (React/Vue) with separate build step
3. Server-rendered templates (html/template)
4. External web server (nginx) with Go backend only

## Decision

Option 1: Single embedded HTML file (<50KB) served by Go's net/http with gorilla/websocket for real-time communication.

### Authentication

- Cookie-based with HMAC-SHA256 session tokens
- HttpOnly, SameSite=Strict cookies (no Secure flag — local/LAN use)
- Session IDs from crypto/rand (16 bytes)
- NO tokens in query strings or client-side JavaScript
- Empty auth_token config = no auth required

**Rejected**: JWT tokens (overhead, unnecessary for single-user), query string tokens (security risk — appears in logs/referrers).

### Rate Limiting

- Per-session: 10 msg/s (token bucket)
- Global: 100 msg/s (token bucket)
- NO per-IP tracking

**Rejected**: Per-IP rate limiting (CTO review round 2: memory leak risk for local deployment, unnecessary complexity).

### Session Management

- Max 50 active sessions
- Cleanup goroutine every 60s
- Configurable max age (default 24h)

### WebSocket

- Read/write buffers: 4096 bytes each (not default 4KB+)
- Max message size: 512KB
- Ping/pong: 60s timeout, 54s ping interval
- Per-session send queue: 100 messages max

### Dashboard

- On-demand status via WebSocket request/response
- NO fixed-interval broadcast (CTO review round 2: wasteful for local use)

### Frontend

- Single HTML file, zero external dependencies
- System fonts only (no web fonts)
- XSS prevention: HTML escape first, then regex whitelist for markdown
- Mobile responsive (320px minimum viewport)
- ARIA accessibility attributes

## Consequences

### Positive

- ~1.7MB estimated memory for 50 concurrent WebSocket connections
- Zero build tooling for frontend
- Embedded in binary — single-file deployment
- No CORS issues (same origin)

### Negative

- Limited UI capabilities without a framework
- No hot-reload during frontend development
- HTML file maintenance is manual

### Risks

- No TLS (mitigated: reverse proxy documentation provided)
- Cookie not Secure (mitigated: local/LAN only deployment model)
