# PicoClaw WhatsApp Bridge

A Node.js bridge that connects WhatsApp Web to PicoClaw via WebSocket, using the [Baileys](https://github.com/WhiskeySockets/Baileys) library (v7).

## Quick Start

```bash
# 1. Install dependencies
cd tools/whatsapp-bridge
npm install

# 2. Start the bridge
node index.js

# 3. Scan QR code with your WhatsApp app
#    Open WhatsApp → Settings → Linked Devices → Link a Device

# 4. Configure PicoClaw (in ~/.picoclaw/config.json)
# "whatsapp": { "enabled": true, "bridge_url": "ws://localhost:3001" }

# 5. Start PicoClaw gateway
picoclaw gateway
```

## Requirements

- Node.js >= 18
- WhatsApp account (personal number)

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `BRIDGE_PORT` | `3001` | WebSocket server port |

PicoClaw config (`~/.picoclaw/config.json`):

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    }
  }
}
```

## How It Works

```
WhatsApp ← Baileys v7 (this bridge) → WebSocket :3001 → PicoClaw WhatsAppChannel
```

1. The bridge connects to WhatsApp Web using the Baileys library
2. It runs a WebSocket server on port 3001
3. PicoClaw's WhatsApp channel connects as a WebSocket client
4. Messages are relayed bidirectionally between WhatsApp and PicoClaw

### Supported Message Types

| Type | Inbound | Outbound |
|------|---------|----------|
| Text | Yes | Yes |
| Images | Caption only | No |
| Video | Caption only | No |
| Documents | Placeholder | No |
| Audio | Placeholder | No |
| Stickers | Placeholder | No |

## Session Persistence

Authentication data is stored in `./auth_store/`. After the first QR scan, the bridge will reconnect automatically without requiring another scan.

To re-authenticate, delete the `auth_store` directory and restart.

## Troubleshooting

**QR code not showing**: Make sure the terminal supports Unicode. Try a different terminal emulator.

**"Couldn't link device"**: Update WhatsApp on your phone to the latest version. Delete `./auth_store/` and restart the bridge to get a fresh QR code.

**"Logged out" error**: WhatsApp revoked the linked device. Delete `./auth_store/` and restart to scan a new QR code.

**Connection drops**: The bridge auto-reconnects with exponential backoff. If PicoClaw is not connected, the bridge will wait for it.

**"bad decrypt" errors on startup**: These are harmless — Baileys is syncing state from scratch. Messages will work normally.

**Error 405 on connection**: Try updating Baileys (`npm install @whiskeysockets/baileys@latest`). This error can also occur when connecting from datacenter IPs.

**Permission on auth_store**: The `auth_store` directory contains session credentials. Ensure it has restricted permissions:
```bash
chmod -R 700 ./auth_store
```

## Protocol

See [PROTOCOL.md](./PROTOCOL.md) for the message format specification.

## Dependencies

- [@whiskeysockets/baileys](https://github.com/WhiskeySockets/Baileys) v7 — WhatsApp Web API
- [ws](https://github.com/websockets/ws) — WebSocket server
- [qrcode-terminal](https://www.npmjs.com/package/qrcode-terminal) — QR code display in terminal
