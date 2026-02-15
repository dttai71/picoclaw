# Testing Strategy

Testing approach for PicoClaw aligned with SDLC 6.0.5 Section 8 (PROFESSIONAL tier).

## Principles

1. **Test at system boundaries** — Focus on public APIs, not internal helpers
2. **Table-driven tests** — Go standard pattern for multiple inputs
3. **No mocks for simple types** — Use real implementations where feasible
4. **Memory-conscious** — Tests must not allocate excessive memory (target hardware: <10MB RAM)
5. **Cross-platform** — Tests must pass on linux/{amd64,arm64,riscv64}, darwin/arm64

## Test Categories

### Unit Tests

| Package | Focus | Priority |
|---------|-------|----------|
| `pkg/agent` | Agent loop, tool iteration, context building | P0 |
| `pkg/auth` | OAuth flow, token storage, permission checks | P0 |
| `pkg/tools` | Tool execution, SSRF protection, path validation | P0 |
| `pkg/channels` | Web auth, rate limiting, session management | P0 |
| `pkg/providers` | Provider selection, request formatting | P1 |
| `pkg/session` | File persistence, concurrent access | P1 |
| `pkg/config` | Config loading, env override, defaults | P1 |
| `pkg/state` | State persistence, concurrent read/write | P2 |
| `pkg/logger` | Log formatting, file permissions | P2 |

### Security Tests

| Test | Description | Package |
|------|-------------|---------|
| SSRF blocking | Internal IP detection for loopback, private, link-local | `pkg/tools` |
| Path traversal | Workspace escape prevention via `../`, symlinks, prefix collision | `pkg/tools` |
| Config permissions | `0600` file mode for sensitive files | `pkg/config` |
| Session permissions | `0600` file mode for session data | `pkg/session` |
| Cookie auth | HMAC-SHA256 validation, HttpOnly, SameSite | `pkg/channels` |
| Rate limiting | Per-session and global token bucket | `pkg/channels` |

### Integration Tests (Future)

| Test | Description | Priority |
|------|-------------|----------|
| Gateway startup | Start gateway, connect Web channel, send/receive message | P1 |
| WhatsApp bridge | Bridge WebSocket protocol compliance | P2 |
| Multi-channel | Multiple channels active simultaneously | P2 |

## Coverage Targets

| Tier | Target | Timeline |
|------|--------|----------|
| Current baseline | 31.1% | 2026-02-15 |
| Sprint 1 target | 40% | +2 weeks |
| Sprint 2 target | 50% | +4 weeks |
| Phase 1 exit | 50% | 5 weeks |
| Phase 2 target | 70% | +3 months |

## Running Tests

```bash
# All tests
make test

# Single package
go test -v ./pkg/channels/...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

## CI Integration

Tests run automatically on:
- Every push to any branch
- Every PR targeting `main`
- Pre-merge check (required to pass)

CI steps: `go generate` → `make fmt` (check diff) → `go vet` → `go test`
