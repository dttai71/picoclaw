# Quality Gates

Quality gates aligned with SDLC 6.0.5 Pillar 4 for PicoClaw (PROFESSIONAL tier).

## Gate Definitions

### G0.1 — Foundation Exit

**Stage**: 00-Foundation → 01-Planning
**Approver**: CTO

| Criteria | Metric | Status |
|----------|--------|--------|
| Vision documented | README.md exists | Pass |
| Architecture constraints defined | <10MB RAM, <10MB binary, <1s boot | Pass |
| Target platforms identified | RISC-V, ARM64, x86_64 | Pass |

### G0.2 — Planning Exit

**Stage**: 01-Planning → 02-Design
**Approver**: CTO

| Criteria | Metric | Status |
|----------|--------|--------|
| Requirements documented | CLAUDE.md architecture section | Pass |
| User stories identified | Channel integrations, tools, agent loop | Pass |
| Scope bounded | Phase 1 features defined | Pass |

### G1 — Design Exit

**Stage**: 02-Design → 03-Integrate/04-Build
**Approver**: CTO

| Criteria | Metric | Status |
|----------|--------|--------|
| System architecture documented | Package diagram in CLAUDE.md | Pass |
| ADRs for key decisions | ADR-001 Web Channel | Pass |
| API contracts defined | WebSocket protocol, Bridge protocol | Pass |
| Security architecture reviewed | SECURITY.md threat model | Pass |

### G2 — Build Exit

**Stage**: 04-Build → 05-Test
**Approver**: CTO + CI

| Criteria | Metric | Status |
|----------|--------|--------|
| Code compiles | `make build` succeeds | Pass |
| `go vet` clean | Zero warnings | Pass |
| `go fmt` clean | No diff | Pass |
| Cross-compilation | `make build-all` succeeds | Pass |
| Binary size | < 10MB | Pass |

### G3 — Test Exit

**Stage**: 05-Test → 06-Deploy
**Approver**: CTO + CI

| Criteria | Metric | Target |
|----------|--------|--------|
| All tests pass | `go test ./...` | 0 failures |
| Overall coverage | `go tool cover` | >= 50% |
| Critical path coverage | agent, auth, tools, channels | >= 60% |
| Security tests pass | SSRF, path traversal, auth | All pass |
| No known P0 vulnerabilities | Security audit | 0 open |

### G4 — Deploy Exit

**Stage**: 06-Deploy → 07-Operate
**Approver**: CTO

| Criteria | Metric | Target |
|----------|--------|--------|
| Release artifacts built | goreleaser | All platforms |
| CHANGELOG updated | Version entry exists | Current version |
| Binary tested on target | Gateway starts successfully | Pass |

## Sprint Gates

### G-Sprint — Sprint Start

| Criteria | Description |
|----------|-------------|
| Backlog groomed | Stories estimated and prioritized |
| Dependencies identified | External deps approved by CTO |
| AGENTS.md current | Reflects latest architecture |

### G-Sprint-Close — Sprint End

| Criteria | Description |
|----------|-------------|
| All stories tested | Unit tests for new code |
| Coverage not decreased | >= previous sprint baseline |
| Documentation updated | Affected docs reflect changes |
| `make test` passes | CI green |

## Enforcement

- **CI Pipeline**: `make fmt`, `go vet`, `go test` run on every PR
- **Coverage tracking**: Baseline recorded per sprint in this document
- **Gate reviews**: CTO approval required for G1+ gates
- **Vibecoding Index**: Target < 20 (currently measuring baseline)

## Coverage History

| Date | Overall | Notes |
|------|---------|-------|
| 2026-02-15 | 31.1% | Initial baseline after Web Channel + WhatsApp Bridge |
