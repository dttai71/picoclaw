---
spec_id: DOC-0001
spec_name: "PicoClaw Vision"
spec_version: "1.0.0"
status: approved
tier: PROFESSIONAL
stage: "00"
category: functional
owner: "CTO"
created: 2026-02-15
last_updated: 2026-02-15
tags: ["vision", "foundation", "strategy"]
---

# PicoClaw Vision

## Vision Statement

**Ultra-lightweight personal AI assistant accessible to everyone, running on $10 hardware.**

PicoClaw brings AI capabilities to resource-constrained devices — from RISC-V dev boards to Raspberry Pi to old laptops — with a single binary that boots in under 1 second using less than 10MB RAM.

## Problem Statement

Existing AI assistant tools (ChatGPT, Claude Desktop, etc.) require:
- Expensive hardware (modern CPU, 8GB+ RAM, SSD)
- Cloud-dependent operation (no offline resilience)
- Complex setup (Docker, Python environments, API wrappers)
- Platform lock-in (Windows/macOS only)

People in emerging markets (Vietnam, Southeast Asia, Africa) and maker communities need AI assistants that work on affordable hardware they already own.

## Target Users

| Segment | Need | Platform |
|---------|------|----------|
| **Makers/Hobbyists** | AI on dev boards (RISC-V, ARM) | MaixCAM, Pi |
| **Students** | Affordable AI tutoring | Old laptops, cheap phones |
| **Small Businesses** | Customer service automation | Zalo, LINE, Telegram |
| **IoT Developers** | AI-powered device control | I2C/SPI interfaces |
| **Vietnamese Market** | AI via Zalo (74M users) | Zalo OA integration |

## Platform Philosophy

| Principle | Target | Rationale |
|-----------|--------|-----------|
| Binary size | <10MB | Fit on minimal storage |
| RAM usage | <10MB | Run on $10 boards |
| Boot time | <1 second | Responsive experience |
| Dependencies | Minimal | Small attack surface |
| Platforms | linux/{amd64,arm64,riscv64}, darwin/arm64, windows/amd64 | Universal |
| Channels | 10+ (Telegram, Discord, LINE, Zalo, Web...) | Meet users where they are |

## Multi-Channel Strategy

PicoClaw reaches users through their preferred messaging platforms:

- **Global**: Telegram, Discord, Slack, WhatsApp, Web
- **Japan/Thailand**: LINE
- **China**: Feishu, DingTalk, QQ (OneBot)
- **Vietnam**: Zalo (OA channel - Phase 3)
- **Hardware**: MaixCAM, I2C/SPI devices

Each channel follows the same pattern: webhook/polling for receiving, REST API for sending, unified through the internal MessageBus.

---

**Last Updated**: 2026-02-15
**Owner**: CTO
