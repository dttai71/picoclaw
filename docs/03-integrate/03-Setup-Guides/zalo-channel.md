---
spec_id: GUIDE-0002
spec_name: "Zalo Channel Setup Guide"
spec_version: "1.0.0"
status: draft
tier: PROFESSIONAL
stage: "03"
category: integration
owner: "PM/PJM"
created: 2026-02-15
last_updated: 2026-02-15
related_adrs: ["ADR-002"]
related_specs: ["SPEC-0003"]
tags: ["zalo", "setup", "guide", "oauth"]
---

# Zalo Channel

The Zalo channel connects PicoClaw to a Zalo Official Account (OA), enabling AI-powered messaging for Vietnamese users.

## Prerequisites

1. A Zalo Official Account — create at [https://oa.zalo.me](https://oa.zalo.me)
2. A Zalo App registered at [https://developers.zalo.me](https://developers.zalo.me)
3. App must have "Send and receive messages" permission approved
4. Obtain: `app_id`, `app_secret`, `oa_secret_key` from the Zalo developer dashboard

## Setup

### Step 1: Configure Credentials

Add to `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "zalo": {
      "enabled": true,
      "app_id": "YOUR_ZALO_APP_ID",
      "app_secret": "YOUR_ZALO_APP_SECRET",
      "oa_secret_key": "YOUR_OA_SECRET_KEY",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18792,
      "webhook_path": "/webhook/zalo",
      "allow_from": []
    }
  }
}
```

### Step 2: OAuth Authorization

Run the OAuth flow to obtain access and refresh tokens:

```bash
picoclaw auth login --provider zalo
```

This will:
1. Print an authorization URL
2. Open your browser to the Zalo OAuth consent page
3. After you authorize, tokens are saved to `~/.picoclaw/zalo_tokens.json` (0600 permissions)
4. Print "OAuth successful"

### Step 3: Configure Webhook

In the Zalo OA dashboard ([https://oa.zalo.me](https://oa.zalo.me)):

1. Go to **Settings > Webhook**
2. Set webhook URL: `https://your-domain.com/webhook/zalo`
3. Subscribe to events: `user_send_text`
4. Enter your `oa_secret_key` for signature verification

For local development, use ngrok:

```bash
ngrok http 18792
# Use the HTTPS URL as your webhook URL
```

### Step 4: Start Gateway

```bash
picoclaw gateway
```

You should see: `Zalo channel enabled successfully`

## Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Enable the Zalo channel |
| `app_id` | `""` | Zalo App ID (from developer dashboard) |
| `app_secret` | `""` | Zalo App Secret |
| `oa_secret_key` | `""` | OA Secret Key (for webhook HMAC verification) |
| `webhook_host` | `0.0.0.0` | Webhook listen address |
| `webhook_port` | `18792` | Webhook HTTP port |
| `webhook_path` | `/webhook/zalo` | Webhook URL path |
| `allow_from` | `[]` | Allowed Zalo user IDs (empty = all) |

Environment variable overrides: `PICOCLAW_CHANNELS_ZALO_*`

## Token Management

- **Access token**: valid ~24 hours, refreshed automatically at <30 minutes remaining
- **Refresh token**: valid ~90 days
- **Storage**: `~/.picoclaw/zalo_tokens.json` with 0600 permissions
- **Auto-refresh**: background goroutine checks every 5 minutes
- **Re-authorization**: if refresh token expires, run `picoclaw auth login --provider zalo` again

## Webhook Verification

Zalo sends a signature in the `X-ZaloOA-Signature` header. PicoClaw verifies it using HMAC-SHA256 with your `oa_secret_key`. Requests with invalid signatures are rejected (HTTP 401).

## Reverse Proxy

For production, use a reverse proxy with TLS:

```nginx
server {
    listen 443 ssl;
    server_name picoclaw.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /webhook/zalo {
        proxy_pass http://127.0.0.1:18792;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "zalo app_id and oa_secret_key required" | Set `app_id` and `oa_secret_key` in config |
| "No stored tokens, OAuth required" | Run `picoclaw auth login --provider zalo` |
| Webhook returns 401 | Verify `oa_secret_key` matches Zalo dashboard |
| Token refresh fails | Check internet connectivity; re-run OAuth if refresh token expired |
| Messages not received | Verify webhook URL in Zalo dashboard, check ngrok if local |

## Limitations (Phase 1)

- Text messages only (image/file support planned for Phase 2)
- 1:1 messaging only (Zalo OA does not support group chats)
- Single OA per PicoClaw instance
