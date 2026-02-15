# Stage 09 - Govern

**Status**: REQUIRED (PROFESSIONAL tier)
**Coverage**: 15% → Target 50%

## Overview

Compliance, security, and strategic oversight for PicoClaw.

## Documents

| Document | Description | Status |
|----------|-------------|--------|
| [SECURITY.md](SECURITY.md) | Threat model, security controls, file permissions | Active |
| [LICENSE-COMPLIANCE.md](LICENSE-COMPLIANCE.md) | Dependency license audit | Active |

## Governance Checklist

- [x] Threat model documented
- [x] File permissions enforced (0600 for sensitive files)
- [x] SSRF protection implemented
- [x] Path traversal prevention
- [x] XSS prevention in web channel
- [x] Cookie security (HttpOnly, SameSite=Strict)
- [x] Rate limiting (per-session + global)
- [x] Session ID from crypto/rand
- [ ] Dependency vulnerability scan (scheduled)
- [ ] Penetration testing (Phase 2)
