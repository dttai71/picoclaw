# System Architecture

## Overview

PicoClaw is an ultra-lightweight personal AI assistant targeting minimal hardware ($10 boards, <10MB RAM, 1-second boot).

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    cmd/picoclaw/main.go                  │
│         CLI: onboard | agent | gateway | status         │
└────────────────────────┬────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌─────▼─────┐   ┌─────▼─────┐
    │  Agent  │    │  Gateway  │   │  Channels │
    │  Loop   │    │  (daemon) │   │  Manager  │
    └────┬────┘    └─────┬─────┘   └─────┬─────┘
         │               │               │
         │          ┌────▼────┐          │
         │          │   Bus   │◄─────────┘
         │          │ (pubsub)│
         │          └────┬────┘
         │               │
    ┌────▼───────────────▼────┐
    │       Providers         │
    │  HTTP | Claude | Codex  │
    │  GitHub Copilot         │
    └─────────────────────────┘
```

## Core Packages

| Package | Responsibility | Memory Budget |
|---------|---------------|---------------|
| `agent` | LLM loop, tool iteration, context | ~2MB |
| `providers` | LLM API abstraction | ~1MB |
| `tools` | Filesystem, shell, web, cron, I2C/SPI | ~1MB |
| `channels` | Telegram, Discord, Slack, Web, WhatsApp, LINE, Zalo | ~2MB |
| `bus` | Event pub/sub (InboundMessage, OutboundMessage) | <100KB |
| `config` | JSON config + env overrides | <100KB |
| `session` | Conversation history (JSON files) | ~500KB |

## Data Flow

### CLI Mode

```
User input → AgentLoop.ProcessDirect() → build context → LLM → tool calls → response
```

### Gateway Mode

```
Channel message → MessageBus.Publish(InboundMessage)
  → AgentLoop.Run() processes event
  → LLM → tool calls → response
  → MessageBus.Publish(OutboundMessage)
  → Channel.Send()
```

## Constraints

| Constraint | Value | Rationale |
|-----------|-------|-----------|
| Binary size | <10MB | Target: $10 hardware |
| RAM usage | <10MB | RISC-V boards |
| Boot time | <1 second | Responsive experience |
| Platforms | linux/{amd64,arm64,riscv64}, darwin/arm64, windows/amd64 | Portability |
| Dependencies | Minimal | Small binary, less attack surface |

## Security Architecture

See [09-govern/SECURITY.md](../../09-govern/SECURITY.md) for threat model and controls.

Key controls:
- Workspace sandboxing (path traversal prevention)
- SSRF protection (internal IP blocking)
- Cookie-based auth for web channel (HMAC-SHA256)
- OAuth 2.0 token storage for Zalo channel (0600 file permissions)
- File permissions: 0600 for sensitive files
