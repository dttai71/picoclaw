# WhatsApp Bridge Protocol v1

## Overview

The WhatsApp bridge communicates with PicoClaw via WebSocket using JSON messages. The bridge acts as a WebSocket **server** on `ws://localhost:3001`, and PicoClaw's WhatsApp channel connects as a **client**.

## Message Format

All messages include:
- `v` — Protocol version (currently `"1"`)
- `type` — Message type
- `timestamp` — Unix timestamp in milliseconds

## Message Types

### Incoming Message (Bridge → PicoClaw)

```json
{
  "v": "1",
  "type": "message",
  "timestamp": 1708000000000,
  "id": "ABCDEF123456",
  "from": "84939116006",
  "chat": "84939116006",
  "content": "Hello from WhatsApp",
  "from_name": "John Doe"
}
```

### Outgoing Message (PicoClaw → Bridge)

```json
{
  "type": "message",
  "to": "84939116006",
  "content": "Response from PicoClaw"
}
```

### Status Update (Bridge → PicoClaw)

```json
{
  "v": "1",
  "type": "status",
  "timestamp": 1708000000000,
  "status": "connected"
}
```

Status values: `connected`, `disconnected`, `qr_required`

When `status` is `qr_required`, an additional `qr` field contains the QR code string.

### Error (Bridge → PicoClaw)

```json
{
  "v": "1",
  "type": "error",
  "timestamp": 1708000000000,
  "code": "WHATSAPP_DISCONNECTED",
  "message": "Session ended by remote",
  "retry_after": 30
}
```

Error codes:
- `WHATSAPP_DISCONNECTED` — WhatsApp connection lost (will auto-reconnect)
- `QR_REQUIRED` — Need to scan QR code again
- `AUTH_FAILED` — WhatsApp account logged out
- `SEND_FAILED` — Failed to send outgoing message
- `RATE_LIMITED` — Too many messages

## Connection Flow

1. Bridge starts WebSocket server on port 3001
2. PicoClaw connects as client
3. Bridge sends current WhatsApp status
4. Bridge connects to WhatsApp Web (or shows QR code)
5. Messages flow bidirectionally

## Health Check

The bridge sends WebSocket ping frames every 54 seconds. PicoClaw responds with pong (handled automatically by gorilla/websocket).
