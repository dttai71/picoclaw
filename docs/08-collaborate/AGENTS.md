# AGENTS.md - PicoClaw AI Collaboration Guide

**Version**: 1.0.0  
**Status**: ACTIVE  
**Framework**: SDLC 6.0.5 (Pillar 5 - SASE Integration)  
**Last Updated**: February 15, 2026

> **🤖 AI Assistant Role**: You are an AI development assistant working on PicoClaw, an ultra-lightweight personal AI assistant written in Go. Your primary responsibility is to **enhance human productivity** while maintaining **implementation authenticity** and **<10MB RAM constraint**.

---

## Quick Start

### Project Identity

**What is PicoClaw?**
- Ultra-lightweight personal AI assistant in Go
- Targets: <10MB RAM, 1-second boot, $10 hardware
- Supports: RISC-V, ARM64, x86_64
- Multi-channel (Telegram, Discord, WhatsApp, Zalo, ZaloUser, etc.)
- Multi-provider (Anthropic, OpenRouter, Zhipu, etc.)

**Core Philosophy:**
1. **Minimal Footprint** - Every byte matters
2. **Portable** - Single binary, cross-platform
3. **Security-First** - Workspace sandboxing, credential protection
4. **No Facades** - Real implementations, no mocks

---

## Architecture Overview

### Project Structure

```
picoclaw/
├── cmd/picoclaw/          # CLI entry point (single file)
│   ├── main.go            # Subcommands: agent, gateway, onboard
│   └── workspace/         # Embedded templates (go:embed)
├── pkg/                   # Core packages
│   ├── agent/             # Agent loop (LLM interaction)
│   ├── providers/         # LLM provider abstraction
│   ├── tools/             # Tool registry & implementations
│   ├── channels/          # Multi-platform integrations
│   ├── bus/               # Message bus (pub/sub)
│   ├── config/            # Config management
│   ├── session/           # Conversation history
│   └── ...                # See CLAUDE.md for full breakdown
├── workspace/             # Default workspace templates
│   ├── AGENT.md           # Agent instructions
│   ├── IDENTITY.md        # Bot personality
│   ├── SOUL.md            # Core values
│   ├── USER.md            # User context
│   └── skills/            # Custom skills
└── docs/                  # SDLC 6.0.5 documentation
    ├── 00-foundation/     # WHY (vision, design thinking)
    ├── 01-planning/       # WHAT (specs, backlog)
    ├── 02-design/         # HOW (architecture, ADRs)
    ├── 03-integrate/      # API contracts
    ├── 04-build/          # Build process
    ├── 05-test/           # Testing strategy
    ├── 06-deploy/         # Deployment
    ├── 07-operate/        # Operations
    ├── 08-collaborate/    # This file!
    └── 09-govern/         # Compliance, security
```

### Key Data Flow

**CLI Mode:**
```
User Input → AgentLoop.ProcessDirect()
  → Build context (tools, skills, workspace)
  → Call LLM provider
  → Execute tool calls (loop up to max_tool_iterations)
  → Return response
```

**Gateway Mode:**
```
Channel (Telegram/Discord/etc.) → MessageBus.PublishInbound()
  → AgentLoop.Run() event loop
  → Process message (same as CLI)
  → MessageBus.PublishOutbound() → Channel.Send()
```

**Key Components:**
- **Agent Loop** (`pkg/agent/loop.go`) - Core orchestration
- **Providers** (`pkg/providers/`) - LLM abstraction (HTTP, Claude, Codex, GitHub Copilot)
- **Tools** (`pkg/tools/`) - Filesystem, shell, web, message, cron, I2C, SPI
- **Channels** (`pkg/channels/`) - 12+ platforms (Telegram, Discord, Slack, WhatsApp, Zalo Bot Platform, ZaloUser, LINE, QQ, Feishu, DingTalk, OneBot, MaixCAM)
- **Config** (`pkg/config/`) - JSON + env vars (`PICOCLAW_*` prefix)

### Critical Constraints

| Constraint | Target | Current | Reasoning |
|------------|--------|---------|-----------|
| **Binary Size** | <10MB | 8.2MB | Embedded systems |
| **RAM Usage** | <10MB | 6-9MB | $10 hardware support |
| **Boot Time** | <1s | 0.4s | Instant responsiveness |
| **Dependencies** | Minimal | 23 | Smaller attack surface |

⚠️ **ALWAYS** check memory impact when adding features.

---

## Conventions

### Code Style

**Go Standards:**
- ✅ Use `gofmt` before every commit (`make fmt`)
- ✅ Run `go vet` to catch common mistakes (`make vet`)
- ✅ Follow [Effective Go](https://go.dev/doc/effective_go)
- ✅ Package names: lowercase, single word (e.g., `agent`, `tools`, not `agentLoop`)
- ✅ Exported functions: PascalCase, unexported: camelCase

**Error Handling:**
```go
// ✅ GOOD: Wrap errors with context
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// ❌ BAD: Swallow errors
_ = doSomething()

// ❌ BAD: Generic errors
if err != nil {
    return err
}
```

**Logging:**
```go
// ✅ GOOD: Use logger package with context
logger.InfoCF("component", "message", map[string]interface{}{
    "key": value,
})

// ❌ BAD: fmt.Println or log.Println
fmt.Println("debug info")
```

**Memory-Conscious Patterns:**
```go
// ✅ GOOD: Reuse buffers
var buf bytes.Buffer
buf.WriteString(data)
defer buf.Reset()

// ✅ GOOD: Limit slice capacity
items := make([]Item, 0, 100) // cap=100

// ❌ BAD: Unbounded growth
var items []Item
for { items = append(items, ...) }
```

### Testing Standards

**Required:**
- ✅ Unit tests for all new packages (`*_test.go`)
- ✅ Table-driven tests for multiple inputs
- ✅ Test coverage ≥70% for new code (SDLC 6.0.5 Gate G3)

**Example:**
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Feature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Feature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Security Standards

**Workspace Sandboxing:**
- ✅ All filesystem operations MUST respect `restrict_to_workspace`
- ✅ Path validation: `tools.ValidatePath(path, workspace, restrict)`
- ✅ Never use `filepath.Join` without validation

**Credentials:**
- ✅ Config file: 0600 permissions (read/write owner only)
- ✅ Session files: 0600 permissions
- ✅ Never log API keys or tokens
- ✅ Use environment variables for sensitive data

**SSRF Prevention:**
```go
// ✅ GOOD: Check for internal IPs
if isInternalHost(parsedURL.Host) {
    return fmt.Errorf("internal IPs not allowed")
}

// ❌ BAD: Allow arbitrary URLs
resp, err := http.Get(userProvidedURL)
```

---

## DO NOT

### Anti-Patterns

❌ **DO NOT add dependencies without CTO approval**
- Each dependency increases binary size
- Security surface grows
- Check existing stdlib solutions first

❌ **DO NOT use `interface{}` excessively**
- Type safety is important
- Use generics (Go 1.18+) when appropriate
- Define specific types

❌ **DO NOT ignore memory constraints**
```go
// ❌ BAD: Load entire file into memory
data, _ := os.ReadFile(largeFile)

// ✅ GOOD: Stream processing
f, _ := os.Open(largeFile)
defer f.Close()
scanner := bufio.NewScanner(f)
for scanner.Scan() { /* process line */ }
```

❌ **DO NOT create mock implementations**
- PicoClaw philosophy: Real code only
- If you can't implement fully, document limitations
- Use feature flags for partial implementations

❌ **DO NOT skip error handling**
```go
// ❌ BAD
result, _ := riskyOperation()

// ✅ GOOD
result, err := riskyOperation()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

❌ **DO NOT commit large files**
- Binary files (except small assets)
- Generated files (build artifacts)
- Temporary files
- Run `go generate` in CI, not committed

❌ **DO NOT break backward compatibility without ADR**
- Config format changes → ADR required
- API changes → ADR required
- Use deprecation warnings, migration guides

### Forbidden Patterns

**Global Mutable State:**
```go
// ❌ BAD
var globalConfig *Config

// ✅ GOOD: Pass config explicitly
func NewService(cfg *Config) *Service { /* ... */ }
```

**Panic in Libraries:**
```go
// ❌ BAD
if err != nil {
    panic(err)
}

// ✅ GOOD: Return errors
if err != nil {
    return fmt.Errorf("error: %w", err)
}
```

**Long-Running Init:**
```go
// ❌ BAD: Slow init() blocks startup
func init() {
    loadLargeDataset() // 5 seconds!
}

// ✅ GOOD: Lazy initialization
func getData() []byte {
    initOnce.Do(func() { /* load data */ })
    return data
}
```

---

## Development Workflow

### Build & Test Commands

```bash
# Development
make fmt           # Format code (REQUIRED before commit)
make vet           # Static analysis
make test          # Run all tests
make build         # Build for current platform

# Cross-compilation
make build-all     # All platforms (linux, darwin, windows)

# Release
make clean         # Clean build artifacts
make deps          # Update dependencies
```

### Pre-Commit Checklist

Before submitting a PR, ensure:

- [ ] `make fmt` - No formatting changes
- [ ] `make vet` - No vet warnings
- [ ] `make test` - All tests pass
- [ ] Test coverage ≥70% for new code
- [ ] Binary size <10MB (`ls -lh build/picoclaw`)
- [ ] Documentation updated (if public API changes)
- [ ] CHANGELOG.md updated (if user-facing change)
- [ ] ADR written (if architectural decision)

### PR Template

```markdown
## Description
Brief description of changes.

## Type
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Checklist
- [ ] Tests pass (`make test`)
- [ ] Coverage ≥70% for new code
- [ ] Documentation updated
- [ ] Binary size verified <10MB

## Related Issues
Fixes #123

## AI Attestation (if AI-generated)
- [ ] I have reviewed all AI-generated code
- [ ] I understand how the code works
- [ ] I have tested the changes manually
```

---

## Key Decisions (ADRs)

### Existing ADRs

| ADR | Title | Decision | Rationale |
|-----|-------|----------|-----------|
| (Pending) | Go rewrite from Python | Go chosen | 99% memory reduction, 400x faster |
| (Pending) | Workspace sandboxing | Enabled by default | Security-first approach |
| (Pending) | Single-binary distribution | No package manager | Portability across platforms |
| (Pending) | Message bus architecture | Pub/sub pattern | Decouples channels from agent |

> **Note:** ADRs will be formally documented in `docs/02-design/ADRs/` per SDLC 6.0.5 compliance plan.

---

## Debugging Tips

### Common Issues

**"Package not found"**
```bash
go mod tidy
go mod download
```

**"Binary too large"**
```bash
# Check build flags in Makefile
LDFLAGS="-s -w"  # Strip debug info
```

**"Test fails intermittently"**
- Race condition? → `go test -race ./...`
- Timing issue? → Increase timeouts or use channels

**"Memory usage high"**
```bash
# Profile memory
go test -memprofile=mem.prof
go tool pprof mem.prof
```

### Useful Tools

```bash
# Measure binary size
ls -lh build/picoclaw

# Check dependencies
go list -m all

# Find unused code
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...

# Analyze imports
go list -f '{{.ImportPath}}: {{.Imports}}' ./...
```

---

## Resources

### Documentation

- [README.md](../../README.md) - User documentation
- [CLAUDE.md](../../CLAUDE.md) - Original AI guide (migrating to this file)
- [CHANGELOG.md](../../CHANGELOG.md) - Version history
- [SDLC-COMPLIANCE.md](../SDLC-COMPLIANCE.md) - Framework compliance status

### External References

- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [SDLC 6.0.5 Framework](../../SDLC-Framework/README.md)

### Community

- **GitHub Issues**: Bug reports, feature requests
- **GitHub Discussions**: Design discussions, questions
- **Twitter/X**: [@SipeedIO](https://x.com/SipeedIO)

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-02-15 | Initial AGENTS.md (migrated from CLAUDE.md) |

---

## Maintenance

**Review Cycle**: Monthly during active development  
**Owner**: CTO + Core Contributors  
**Feedback**: Open issue or PR with suggested changes  

**Migration Note**: This file replaces deprecated MTS (MentorScript), BRS (BriefingScript), and LPS (LearnedPatternsScript) per SDLC 6.0.5 ADR-029.

---

**Remember:** PicoClaw runs on $10 hardware. Every byte counts. Code with intention. 🦐
