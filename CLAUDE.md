# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PicoClaw is an ultra-lightweight personal AI assistant written in Go. It targets minimal hardware ($10 boards, <10MB RAM, 1-second boot) and supports RISC-V, ARM64, and x86_64. Inspired by [nanobot](https://github.com/HKUDS/nanobot).

## Build & Development Commands

```bash
make build          # Build for current platform (output: build/picoclaw)
make build-all      # Cross-compile for linux/{amd64,arm64,riscv64}, darwin/arm64, windows/amd64
make install        # Build and install to ~/.local/bin
make test           # Run all tests
make fmt            # Format Go code (required before commits)
make vet            # Run go vet
make deps           # Update and tidy dependencies
make clean          # Remove build artifacts
```

Run a single package's tests:
```bash
go test -v ./pkg/agent
go test -v ./pkg/tools/...
```

The `build` target first runs `go generate ./...` which copies `workspace/` into `cmd/picoclaw/workspace` for embedding.

## CI Requirements (PR checks)

PRs must pass: `make fmt` (no diff), `go vet ./...`, and `go test ./...`. The CI also runs `go generate ./...` before vet and test.

## Architecture

### Entry Point & CLI

`cmd/picoclaw/main.go` — Single-file CLI with subcommands: `onboard`, `agent`, `gateway`, `status`, `auth`, `migrate`, `cron`, `skills`, `version`. No framework; uses manual `os.Args` parsing.

Workspace templates in `workspace/` are embedded into the binary via `//go:embed` and copied on `picoclaw onboard`.

### Core Packages (pkg/)

| Package | Purpose |
|---------|---------|
| `agent` | Agent loop: receives messages, builds context, calls LLM, executes tool calls in a loop, returns responses |
| `providers` | LLM provider abstraction. `LLMProvider` interface with `Chat()` and `GetDefaultModel()`. Implementations: HTTP (OpenAI-compatible), Claude, Codex, GitHub Copilot |
| `tools` | Tool registry and implementations (filesystem, shell, web search, cron, I2C, SPI, message, spawn) |
| `channels` | Multi-channel integrations: Telegram, Discord, Slack, Feishu, DingTalk, LINE, QQ, OneBot, MaixCAM, WhatsApp (Baileys bridge), Zalo (Bot Platform API), ZaloUser (zca-cli personal) |
| `bus` | Event message bus for pub/sub between agent, channels, and services |
| `config` | Config loading from `~/.picoclaw/config.json` with env var overrides (`PICOCLAW_*` prefix, via `caarlos0/env` struct tags) |
| `session` | Conversation history persistence (JSON files) |
| `cron` | Scheduled job management |
| `heartbeat` | Periodic task execution (reads HEARTBEAT.md from workspace) |
| `skills` | Custom skill loader/installer (SKILL.md files from workspace, global, or GitHub) |
| `voice` | Voice transcription via Groq Whisper |
| `auth` | OAuth and token-based authentication |
| `state` | Persistent state management |
| `devices` | USB device monitoring |
| `migrate` | Migration from OpenClaw |

### Key Data Flow

1. **CLI mode** (`agent`): User message → `AgentLoop.ProcessDirect()` → build context with tools/skills → call LLM → process tool calls in loop → return response
2. **Gateway mode** (`gateway`): Channels receive messages → publish to `MessageBus` → `AgentLoop.Run()` event loop processes them → responses sent back through channels
3. **Heartbeat**: Timer reads `HEARTBEAT.md` → spawns subagents for long tasks → subagents communicate results via `message` tool

### Provider Selection

`providers.CreateProvider()` in `http_provider.go` resolves providers by: explicit config (`agents.defaults.provider`) → model name prefix detection (e.g., "claude" → Anthropic, "glm" → Zhipu, "openrouter/..." → OpenRouter).

### Tool System

Tools implement the `Tool` interface (`Name()`, `Description()`, `Parameters()`, `Execute()`). Optional interfaces: `ContextualTool` (channel/chat context), `AsyncTool` (async callbacks). Registered via `ToolRegistry.Register()`. The agent loop iterates tool calls up to `max_tool_iterations` (default 20).

### Workspace

Default `~/.picoclaw/workspace/` contains: `AGENTS.md`, `IDENTITY.md`, `SOUL.md`, `USER.md`, `HEARTBEAT.md`, `TOOLS.md`, plus `memory/`, `sessions/`, `state/`, `cron/`, `skills/` directories. All templates are embedded in the binary.

## Version Injection

Version, git commit, build time, and Go version are injected via LDFLAGS at build time (see Makefile `LDFLAGS`).

## Config

Config lives at `~/.picoclaw/config.json`. All values can be overridden with `PICOCLAW_*` environment variables. Key sections: `agents.defaults` (model, workspace, max_tokens), `providers` (API keys/bases), `channels` (bot tokens), `tools.web` (search config), `heartbeat`, `gateway`.

## Channel-Specific Notes

### Zalo (Bot Platform API) — `pkg/channels/zalo.go`

- **API**: `https://bot-api.zapps.me/bot{token}/{method}` — Telegram-style REST API (all POST)
- **Token format**: `id:secret` from [Zalo Bot Creator](https://zalo.me/s/botcreator/) — permanent, no OAuth needed
- **Modes**: Long-polling (default, no public URL) or webhook (requires HTTPS)
- **Polling**: `getUpdates` with 30s timeout, exponential backoff on errors, 408 = no updates (not an error)
- **Webhook**: `X-Bot-Api-Secret-Token` header verification, 1MB body limit, async processing
- **API response format**: Dual envelope format — `{ok, result, error_code}` or `{error, message, data}`
- **getUpdates quirk**: API returns single update object (not array); code handles both via `json.RawMessage`
- **Offset tracking**: Uses `message.date` (unix timestamp) since API has no `update_id` field
- **Events**: `message.text.received`, `message.image.received`, `message.sticker.received`
- **Message fields**: Sender at `message.from.id`, chat at `message.chat.id`, text at `message.text`
- **Outbound**: `sendMessage` with `{chat_id, text}`, text chunked at 2000 chars with 500ms rate limit
- **Config**: `channels.zalo.token`, `channels.zalo.mode` ("polling" or "webhook")
- **Validation**: Token must contain `:` separator; `getMe()` called on Start to verify

### ZaloUser (Personal Account via zca-cli) — `pkg/channels/zalouser.go`

- **Pattern**: Wraps external `zca` CLI binary — spawns `zca listen -r -k` as long-running child process
- **Inbound**: Reads JSON lines from stdout via `bufio.Scanner`, parses `zcaMessage` structs
- **Outbound**: `exec.CommandContext(zca, "msg", "send", chatID, text)` — serialized via `sendMu` mutex
- **Message struct**: `{threadId, msgId, type, content, timestamp, metadata: {threadType, senderId}}`
- **Message types**: `type=1` is text (only type handled); `threadType=1` is DM, `2` is group
- **Auto-restart**: Exponential backoff (5s → 5min) when `zca listen` process exits
- **Profile support**: `-p profile` flag passed to all zca commands when `profile` is configured
- **Binary discovery**: `exec.LookPath(zca)` on init — errors if not found
- **Config**: `channels.zalouser.enabled`, `channels.zalouser.profile`, `channels.zalouser.zca_path`

### WhatsApp (Baileys Bridge) — `pkg/channels/whatsapp.go`

- **Architecture**: Node.js bridge (Baileys v7) → WebSocket server :3001 → PicoClaw Go client (gorilla/websocket)
- **Bridge**: `tools/whatsapp-bridge/index.js` — ESM, uses `@whiskeysockets/baileys` v7
- **Protocol**: JSON over WebSocket, protocol v1 — types: `message`, `status`, `error`
- **Inbound message fields**: `{from, chat, content, id, from_name}` — phone numbers without @s.whatsapp.net
- **Outbound format**: `{type: "message", to: "phone_number", content: "text"}`
- **Status events**: `connected`, `disconnected`, `qr_required` (with QR string)
- **Reconnect**: Exponential backoff (1s → 5min), max 10 attempts on connection loss
- **Session persistence**: Auth stored in `tools/whatsapp-bridge/auth_store/` — survives restarts
- **First run**: QR code displayed in bridge terminal — scan with WhatsApp → Linked Devices
- **Known issues**: "bad decrypt" errors on fresh connection are harmless (Baileys state sync)
- **Bridge health**: Ping frames every 54s, auto-handled by gorilla/websocket
- **Config**: `channels.whatsapp.bridge_url` (default: `ws://localhost:3001`)
