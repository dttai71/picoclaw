# License Compliance

Dependency license audit for PicoClaw. Last updated: 2026-02-15.

## Direct Dependencies

| Package | Version | License | Risk |
|---------|---------|---------|------|
| github.com/adhocore/gronx | v1.19.6 | MIT | Low |
| github.com/anthropics/anthropic-sdk-go | v1.22.1 | MIT | Low |
| github.com/bwmarrin/discordgo | v0.29.0 | BSD-3 | Low |
| github.com/caarlos0/env/v11 | v11.3.1 | MIT | Low |
| github.com/chzyer/readline | v1.5.1 | MIT | Low |
| github.com/github/copilot-sdk/go | v0.1.23 | MIT | Low |
| github.com/google/uuid | v1.6.0 | BSD-3 | Low |
| github.com/gorilla/websocket | v1.5.3 | BSD-2 | Low |
| github.com/larksuite/oapi-sdk-go/v3 | v3.5.3 | MIT | Low |
| github.com/mymmrac/telego | v1.6.0 | MIT | Low |
| github.com/open-dingtalk/dingtalk-stream-sdk-go | v0.9.1 | MIT | Low |
| github.com/openai/openai-go/v3 | v3.22.0 | Apache-2.0 | Low |
| github.com/slack-go/slack | v0.17.3 | BSD-2 | Low |
| github.com/tencent-connect/botgo | v0.2.1 | MIT | Low |
| golang.org/x/oauth2 | v0.35.0 | BSD-3 | Low |

## WhatsApp Bridge (Node.js)

| Package | Version | License | Risk |
|---------|---------|---------|------|
| @whiskeysockets/baileys | ^6.7.0 | MIT | Low |
| qrcode-terminal | ^0.12.0 | Apache-2.0 | Low |
| ws | ^8.16.0 | MIT | Low |

## Summary

- All direct dependencies use permissive licenses (MIT, BSD, Apache-2.0)
- No copyleft (GPL/LGPL/AGPL) dependencies detected
- No commercial license restrictions
- Risk level: **LOW** across all dependencies

## Policy

- New dependencies require CTO approval (per AGENTS.md)
- Copyleft licenses (GPL/LGPL/AGPL) are not permitted without explicit review
- License audit must be updated when dependencies change
