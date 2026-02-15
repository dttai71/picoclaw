---
spec_id: DOC-0003
spec_name: "PicoClaw Product Roadmap"
spec_version: "1.0.0"
status: approved
tier: PROFESSIONAL
stage: "00"
category: functional
owner: "CTO"
created: 2026-02-15
last_updated: 2026-02-15
tags: ["roadmap", "phases", "planning"]
---

# Product Roadmap

## Phase Overview

```
Phase 1 (Complete) ─── Core + Basic Channels
Phase 2 (Complete) ─── Extended Channels + Web
Phase 3 (Current)  ─── Zalo + SDLC + Security
Phase 4 (Planned)  ─── Voice + IoT + Edge AI
```

## Phase 1: Core Foundation (Complete)

- Agent loop with LLM integration
- CLI mode (`picoclaw agent`)
- Tool system (filesystem, shell, web search, cron)
- Provider abstraction (OpenAI, Claude, Codex)
- Basic channels: Telegram, Discord, Slack
- Gateway mode (`picoclaw gateway`)
- Session persistence
- Workspace management (`picoclaw onboard`)

## Phase 2: Channel Expansion (Complete)

- LINE channel (Japan/Thailand market)
- WhatsApp channel (via bridge)
- Feishu, DingTalk channels (China market)
- QQ channel (OneBot protocol)
- Web channel (embedded HTML + WebSocket)
- MaixCAM hardware integration
- Heartbeat system (scheduled tasks)
- Skills system (custom skill loading)
- Voice transcription (Groq Whisper)

## Phase 3: Vietnam Market + Quality (Current)

| Item | Status | Target Date |
|------|--------|-------------|
| **Zalo OA channel** | Proposed | Feb 22, 2026 |
| SDLC 6.0.5 compliance | In Progress | Feb 28, 2026 |
| Security hardening (Part 0) | Complete | Feb 15, 2026 |
| Web channel (Part 1) | Planned | Mar 2026 |
| WhatsApp bridge (Part 2) | Planned | Mar 2026 |
| Test coverage 31% -> 50% | In Progress | Mar 2026 |

### Zalo Channel Milestones

1. ADR-002 + Stage 00-03 documentation
2. CTO approval gate
3. 5-day implementation sprint
4. Integration testing with Zalo OA test account

## Phase 4: Edge AI + IoT (Planned)

- Voice input/output channels
- On-device inference (GGML/llama.cpp integration)
- Enhanced I2C/SPI tool suite
- Device mesh networking
- Prometheus metrics export
- Docker/container deployment option

---

**Last Updated**: 2026-02-15
**Owner**: CTO
