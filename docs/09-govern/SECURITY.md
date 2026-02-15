# Security

PicoClaw runs on personal machines with access to API keys, conversation history, and shell execution. This document describes the security model and fixes applied.

## Threat Model

PicoClaw is designed for **local/LAN deployment** on a single-user machine. The primary threats are:

1. **Local privilege escalation** — Other users on the same machine reading sensitive files
2. **SSRF** — LLM-directed web fetch tool accessing internal services
3. **Path traversal** — LLM-directed file tools escaping the workspace
4. **XSS** — Malicious content rendered in the web interface

PicoClaw does NOT protect against a compromised LLM provider or a malicious LLM model.

## File Permissions

| File Type | Permission | Rationale |
|-----------|-----------|-----------|
| Config (`config.json`) | `0600` | Contains API keys |
| Session files | `0600` | Contains conversation history |
| Auth tokens (`auth.json`) | `0600` | Contains OAuth tokens |
| Log files | `0640` | May contain sensitive data in debug mode |
| Workspace files | `0644` | User-created content |

Directories use `0755` for traversal.

## Path Validation

The filesystem tools enforce workspace restriction using `filepath.Abs` + `filepath.Clean` + prefix check with path separator to prevent:

- `../` traversal attacks
- Symlink escapes
- Prefix collision (e.g., `/workspace-evil` matching `/workspace`)

## SSRF Protection

The web fetch tool blocks requests to internal IP addresses:

- Loopback: `127.0.0.0/8`, `::1`
- Private: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Link-local: `169.254.0.0/16`, `fe80::/10`
- Zero addresses: `0.0.0.0`, `::`

DNS resolution is checked before making the request to prevent DNS rebinding.

## Web Channel Security

- **Authentication**: Cookie-based with HMAC-SHA256 tokens. HttpOnly, SameSite=Strict.
- **Session IDs**: Generated with `crypto/rand` (16 bytes).
- **Rate limiting**: Per-session (10 msg/s) and global (100 msg/s) token bucket.
- **XSS prevention**: HTML is escaped before markdown rendering. No raw `innerHTML` from user content.
- **Headers**: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`.

## Shell Execution

The shell tool blocks dangerous commands via regex patterns (rm -rf, format, dd, shutdown, fork bombs). Workspace restriction prevents path traversal in shell commands.

## Reporting

If you discover a security vulnerability, please open an issue at the project repository.
