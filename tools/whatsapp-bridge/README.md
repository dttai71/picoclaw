# PicoClaw WhatsApp Bridge

A Node.js bridge that connects WhatsApp Web to PicoClaw via WebSocket.

## Quick Start

```bash
# 1. Install dependencies
npm install

# 2. Start the bridge
node index.js

# 3. Scan QR code with your WhatsApp app (Settings → Linked Devices)

# 4. Configure PicoClaw
# In ~/.picoclaw/config.json:
# "whatsapp": { "enabled": true, "bridge_url": "ws://localhost:3001" }

# 5. Start PicoClaw
picoclaw gateway
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `BRIDGE_PORT` | `3001` | WebSocket server port |

## How It Works

```
WhatsApp ← Baileys (this bridge) → WebSocket :3001 → PicoClaw WhatsAppChannel
```

1. The bridge connects to WhatsApp Web using the Baileys library
2. It runs a WebSocket server on port 3001
3. PicoClaw's WhatsApp channel connects as a WebSocket client
4. Messages are relayed between WhatsApp and PicoClaw

## Session Persistence

Authentication data is stored in `./auth_store/`. After the first QR scan, the bridge will reconnect automatically without requiring another scan.

To re-authenticate, delete the `auth_store` directory and restart.

## Troubleshooting

**QR code not showing**: Make sure the terminal supports Unicode. Try a different terminal emulator.

**"Logged out" error**: WhatsApp revoked the linked device. Delete `./auth_store/` and restart to scan a new QR code.

**Connection drops**: The bridge auto-reconnects with exponential backoff. If PicoClaw is not connected, the bridge will wait for it.

**Permission on auth_store**: The `auth_store` directory contains session credentials. Ensure it has restricted permissions:
```bash
chmod -R 700 ./auth_store
```

## Protocol

See [PROTOCOL.md](./PROTOCOL.md) for the message format specification.
