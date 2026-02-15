---
spec_id: GUIDE-0001
spec_name: "Web Channel Setup Guide"
spec_version: "1.0.0"
status: active
tier: PROFESSIONAL
stage: "03"
category: integration
owner: "PM/PJM"
created: 2026-02-15
last_updated: 2026-02-15
related_adrs: ["ADR-001"]
tags: ["web", "setup", "guide", "websocket", "auth"]
---

# Web Channel

The Web channel provides a browser-based chat interface and dashboard for PicoClaw.

## Setup

Add to `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "web": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18791,
      "auth_token": "",
      "session_max_age": 86400
    }
  }
}
```

Then start PicoClaw:

```bash
picoclaw gateway
```

Open `http://localhost:18791` in your browser.

## Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Enable the web channel |
| `host` | `0.0.0.0` | Listen address |
| `port` | `18791` | HTTP/WebSocket port |
| `auth_token` | `""` | Access token (empty = no auth) |
| `session_max_age` | `86400` | Session lifetime in seconds (24h) |
| `allow_from` | `[]` | Allowed sender IDs (empty = all) |

Environment variable overrides: `PICOCLAW_CHANNELS_WEB_*`

## Authentication

When `auth_token` is set:

1. Browser loads `http://localhost:18791` → sees login form
2. User enters the access token
3. Client sends `POST /auth` with `{"token":"..."}`
4. Server validates and sets an HttpOnly cookie (`picoclaw_session`)
5. Cookie is used for WebSocket upgrade and API calls
6. No tokens in query strings or client-side JavaScript

Cookie security: `HttpOnly`, `SameSite=Strict`, not `Secure` (for local/LAN use).

When `auth_token` is empty, no authentication is required.

## Rate Limiting

- Per-session: 10 messages/second
- Global: 100 messages/second
- Exceeding the limit returns `{"type":"error","message":"rate limit exceeded"}`

## WebSocket Protocol

### Connection

`GET /ws` — Upgrades to WebSocket. Server sends session info:

```json
{"type": "session", "session_id": "abc123..."}
```

### Sending Messages

```json
{"type": "message", "content": "Hello PicoClaw"}
```

### Receiving Responses

```json
{"type": "message", "content": "Response from the agent"}
```

### Dashboard Status

Client sends:
```json
{"type": "request_status"}
```

Server responds:
```json
{
  "type": "status",
  "timestamp": 1708000000,
  "data": {
    "active_connections": 2,
    "uptime_sec": 3600,
    "auth_enabled": true
  }
}
```

### Errors

```json
{"type": "error", "message": "rate limit exceeded"}
```

## Reverse Proxy

For production/remote access, use a reverse proxy with TLS:

```nginx
server {
    listen 443 ssl;
    server_name picoclaw.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:18791;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }
}
```

## Limits

- Max message size: 512KB
- Max active sessions: 50
- Max queued messages per session: 100
- WebSocket ping/pong: 60s timeout, 54s ping interval
