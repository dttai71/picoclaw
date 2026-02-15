---
spec_id: DOC-0002
spec_name: "PicoClaw Business Case"
spec_version: "1.0.0"
status: approved
tier: PROFESSIONAL
stage: "00"
category: functional
owner: "PM/PJM"
created: 2026-02-15
last_updated: 2026-02-15
tags: ["business-case", "market", "zalo"]
---

# Business Case

## Value Proposition

PicoClaw provides AI assistant capabilities on ultra-low-cost hardware with zero cloud dependency for core operations, reaching users through native messaging platforms.

## Market Analysis

### Vietnam Market (Zalo Channel)

| Metric | Value |
|--------|-------|
| Zalo monthly active users | 74M+ |
| Vietnam population | ~100M |
| Zalo penetration | ~74% |
| Competitor presence | Low (no lightweight AI bots on Zalo) |

Zalo is the dominant messaging platform in Vietnam, surpassing Facebook Messenger for local communication. Adding Zalo OA support enables:
- Customer service automation for Vietnamese small businesses
- AI tutoring for Vietnamese students
- Personal AI assistant in native platform

### Regional Expansion

| Region | Platform | Users | Status |
|--------|----------|-------|--------|
| Global | Telegram | 900M+ | Supported |
| Global | Discord | 200M+ | Supported |
| Japan/Thailand | LINE | 200M+ | Supported |
| China | Feishu/DingTalk/QQ | 500M+ | Supported |
| **Vietnam** | **Zalo** | **74M+** | **Phase 3** |

### Cost Analysis

| Resource | Cost |
|----------|------|
| New dependencies | $0 (standard library + existing go.mod) |
| Binary size increase | ~50KB (new Go source) |
| RAM overhead | <500KB per channel |
| External services | $0 (Zalo OA free tier) |
| Maintenance | Low (follows proven LINE pattern) |

## Success Metrics

- Zalo webhook latency <500ms (message receive to agent response)
- Token refresh success rate >99.9%
- Zero regressions in existing channels
- Binary remains <10MB

---

**Last Updated**: 2026-02-15
**Owner**: PM/PJM
