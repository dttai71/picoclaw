# Stage 02: Design

**Version**: 6.0.5
**Stage**: 02 - Design
**Status**: ACTIVE
**Framework**: SDLC 6.0.5 Complete Lifecycle
**Tier**: PROFESSIONAL

---

## Purpose

Define **HOW** it will be built. Architecture, design, and technical specifications.

---

## Folder Structure

```text
02-design/
├── README.md                         # This file (P0 entry point)
├── 01-System-Architecture/           # Package diagram, data flow, constraints
├── 02-Data-Model/                    # Config schema, session format
├── 03-API-Design/                    # WebSocket protocol, Bridge protocol
├── 04-ADRs/                          # Architecture Decision Records
└── ADRs/                             # (Legacy, see 04-ADRs)
```

Legacy content archived in `docs/10-archive/02-Legacy/` per RFC-001 (SDLC 6.0.5).

---

## Key Documents

| Document | Purpose | Status |
|----------|---------|--------|
| [System Architecture](01-System-Architecture/README.md) | Package diagram, data flow, constraints | Active |
| [ADR-001: Web Channel](04-ADRs/ADR-001-web-channel.md) | Web Channel architecture decisions | Accepted |
| [ADR-002: Zalo Channel](04-ADRs/ADR-002-zalo-channel.md) | Zalo Channel architecture decisions | Proposed |

---

## ADR Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [ADR-001](04-ADRs/ADR-001-web-channel.md) | Web Channel Architecture | Accepted | 2026-02-15 |
| [ADR-002](04-ADRs/ADR-002-zalo-channel.md) | Zalo Channel Architecture | Proposed | 2026-02-15 |

---

## AI Assistant Guidance

**DO Read**:
- This README for context
- Key documents listed above
- AGENTS.md for coding conventions

**DO NOT Read**:
- `10-archive/` folder - Contains archived, outdated content

---

**Document Status**: P0 Entry Point
**Compliance**: SDLC 6.0.5 Stage 02
**Last Updated**: 2026-02-15
**Owner**: CTO
