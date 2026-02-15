# Stage 06 - Deploy

**Status**: REQUIRED (PROFESSIONAL tier)
**Coverage**: 75%

## Overview

Release and deployment for PicoClaw.

## Release Process

1. Tag version: `git tag v0.x.y`
2. GoReleaser builds cross-platform binaries
3. GitHub Release with artifacts

## Deployment Methods

| Method | Command | Description |
|--------|---------|-------------|
| Direct install | `make install` | Copies to `~/.local/bin` |
| Binary download | GitHub Releases | Pre-built binaries |
| From source | `make build` | Build locally |

## First-Time Setup

```bash
# 1. Install binary
make install

# 2. Onboard (creates config + workspace)
picoclaw onboard

# 3. Configure API key
# Edit ~/.picoclaw/config.json

# 4. Verify
picoclaw version
picoclaw agent "Hello"
```
