# Operations Playbook

Operations guide for running PicoClaw in production.

## Deployment Modes

### CLI Mode (Agent)

Single-user interactive mode.

```bash
picoclaw agent "your message here"
```

- No persistent process
- Session stored in `~/.picoclaw/workspace/sessions/`
- Exit code 0 on success

### Gateway Mode

Multi-channel daemon with persistent connections.

```bash
picoclaw gateway
```

- Runs as long-lived process
- Manages all enabled channels (Telegram, Discord, Web, WhatsApp, etc.)
- Heartbeat system for scheduled tasks
- Graceful shutdown on SIGINT/SIGTERM

## Configuration

Config file: `~/.picoclaw/config.json` (permissions: 0600)

Environment overrides: `PICOCLAW_*` prefix (e.g., `PICOCLAW_CHANNELS_WEB_PORT=8080`)

### Critical Config Items

| Setting | Default | Description |
|---------|---------|-------------|
| `agents.defaults.model` | varies | LLM model to use |
| `agents.defaults.max_tokens` | 4096 | Max response tokens |
| `agents.defaults.max_tool_iterations` | 20 | Tool call loop limit |
| `gateway.auto_start_channels` | true | Start channels on gateway boot |
| `channels.web.port` | 18791 | Web channel HTTP port |
| `channels.web.auth_token` | "" | Access token (empty = no auth) |

## Health Monitoring

### Gateway Health Check

```bash
picoclaw status
```

Returns: active channels, uptime, connected sessions.

### Web Channel Health

- WebSocket ping/pong every 54s (timeout: 60s)
- Max 50 active sessions
- Rate limit: 10 msg/s per session, 100 msg/s global

### WhatsApp Bridge Health

- Bridge ping/pong every 54s
- Auto-reconnect with exponential backoff (1s → 5min, max 10 attempts)
- Session persistence in `./auth_store/`

## Troubleshooting

### Gateway won't start

1. Check config syntax: `cat ~/.picoclaw/config.json | python3 -m json.tool`
2. Check port availability: `lsof -i :18791`
3. Check permissions: `stat -f "%OLp" ~/.picoclaw/config.json` (should be 600)

### Web channel unreachable

1. Verify enabled: `"web": {"enabled": true}` in config
2. Check firewall: port must be accessible
3. Check logs for bind errors

### WhatsApp bridge disconnects

1. Check `./auth_store/` permissions (should be 700)
2. Delete `./auth_store/` and re-scan QR if auth fails
3. Check bridge process: `node index.js` in `tools/whatsapp-bridge/`

### High memory usage

- Expected: <10MB for gateway with all channels
- Web channel: ~1.7MB for 50 concurrent WebSocket connections
- If exceeding 20MB: check for session leak, restart gateway

## Backup

### What to back up

| Path | Content | Frequency |
|------|---------|-----------|
| `~/.picoclaw/config.json` | API keys, channel tokens | On change |
| `~/.picoclaw/workspace/` | Identity, memory, skills | Weekly |
| `~/.picoclaw/workspace/sessions/` | Conversation history | Optional |

### Restore

```bash
cp backup/config.json ~/.picoclaw/config.json
chmod 600 ~/.picoclaw/config.json
cp -r backup/workspace/ ~/.picoclaw/workspace/
```

## Security Operations

- Config file: 0600 (API keys)
- Session files: 0600 (conversation history)
- Auth tokens: 0600 (OAuth tokens)
- Log files: 0640 (may contain sensitive data in debug mode)
- Workspace files: 0644 (user-created content)

See [09-govern/SECURITY.md](../09-govern/SECURITY.md) for full security model.
