# Setup Guides

## Available Guides

| Guide | Description |
|-------|-------------|
| [web-channel.md](web-channel.md) | Web channel setup, auth, WebSocket protocol, reverse proxy |
| [zalo-channel.md](zalo-channel.md) | Zalo OA channel setup, OAuth, webhook configuration |

## Quick Start

```bash
# 1. Onboard (creates config + workspace)
picoclaw onboard

# 2. Configure provider
# Edit ~/.picoclaw/config.json with your API key

# 3. CLI mode
picoclaw agent "Hello"

# 4. Gateway mode (multi-channel)
picoclaw gateway
```

See individual channel docs in `~/.picoclaw/config.json` for channel-specific setup.
