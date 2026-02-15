# Stage 05 - Test

**Status**: REQUIRED (PROFESSIONAL tier)
**Coverage**: 40% → Target 70%

## Overview

Quality assurance and validation for PicoClaw. Ensures reliability across RISC-V, ARM64, and x86_64 targets with minimal resource usage.

## Documents

| Document | Description | Status |
|----------|-------------|--------|
| [QUALITY-GATES.md](QUALITY-GATES.md) | Quality gate definitions for all stages | Active |
| [TESTING-STRATEGY.md](TESTING-STRATEGY.md) | Testing approach, coverage targets, CI integration | Active |

## Current Metrics

| Package | Coverage | Target |
|---------|----------|--------|
| `pkg/agent` | 50.6% | 70% |
| `pkg/auth` | 44.8% | 70% |
| `pkg/channels` | 5.8% | 50% |
| `pkg/config` | 1.3% | 40% |
| `pkg/heartbeat` | 60.2% | 70% |
| `pkg/logger` | 56.5% | 70% |
| `pkg/migrate` | 63.1% | 70% |
| `pkg/providers` | 46.3% | 60% |
| `pkg/session` | 57.8% | 70% |
| `pkg/state` | 75.5% | 70% |
| `pkg/tools` | 36.1% | 60% |
| **Total** | **31.1%** | **50%** |

Baseline measured: 2026-02-15.
