# Stage 04 - Build

**Status**: REQUIRED (PROFESSIONAL tier)
**Coverage**: 80%

## Overview

Development and implementation for PicoClaw.

## Build System

```bash
make build          # Build for current platform (output: build/picoclaw)
make build-all      # Cross-compile: linux/{amd64,arm64,riscv64}, darwin/arm64, windows/amd64
make install        # Build and install to ~/.local/bin
make fmt            # Format Go code (required before commits)
make vet            # Run go vet
make deps           # Update and tidy dependencies
make clean          # Remove build artifacts
```

## Build Pipeline

1. `go generate ./...` (copies workspace/ into cmd/picoclaw/workspace)
2. `go build` with LDFLAGS (version, commit, build time, Go version)
3. Cross-compilation via `GOOS`/`GOARCH` matrix

## Version Injection

```makefile
LDFLAGS = -X main.Version=$(VERSION) -X main.GitCommit=$(COMMIT) \
          -X main.BuildTime=$(BUILDTIME) -X main.GoVersion=$(GOVERSION)
```

## Artifacts

| Target | Binary | Size Target |
|--------|--------|-------------|
| linux/amd64 | picoclaw | <10MB |
| linux/arm64 | picoclaw | <10MB |
| linux/riscv64 | picoclaw | <10MB |
| darwin/arm64 | picoclaw | <10MB |
| windows/amd64 | picoclaw.exe | <10MB |
