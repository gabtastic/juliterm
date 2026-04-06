# Security Policy

## Reporting a vulnerability

If you discover a security vulnerability in Juli, please report it responsibly.

**Do not open a public issue.**

Instead, email **okyaycihan@gmail.com** with:
- A description of the vulnerability
- Steps to reproduce
- Potential impact

You should receive a response within 48 hours. Once the issue is confirmed, a fix will be developed and released as soon as possible.

## Supported versions

Only the latest release is actively supported with security fixes.

## Security considerations

- Always use a strong, unique `--token` value.
- When using `--tunnel`, your server is exposed to the internet — choose a strong token.
- For persistent remote access, use TLS (`--tls-cert`, `--tls-key`) or a reverse proxy.
- Juli validates WebSocket origins to prevent cross-site hijacking.
