# Remote Access

Juli can expose your tmux sessions to the internet via Cloudflare Quick Tunnel, letting you access your terminal from any device with a browser.

## Quick Start

```bash
juli --token SECRET --tunnel
```

Juli starts locally and creates a Cloudflare Quick Tunnel in the background. Once the tunnel is ready, a public URL and QR code appear in the terminal. Scan the QR code with your phone or copy the URL to any browser.

## Prerequisites

Install `cloudflared`:

- **macOS:** `brew install cloudflared`
- **Debian/Ubuntu:** `apt install cloudflared`
- **Other:** Download from [cloudflare/cloudflared releases](https://github.com/cloudflare/cloudflared/releases)

No Cloudflare account is needed for Quick Tunnels.

## How It Works

The `--tunnel` flag tells Juli to spawn `cloudflared tunnel --url http://localhost:<port>` as a child process. Cloudflare assigns a random `*.trycloudflare.com` URL that proxies traffic to your local Juli instance. The URL changes every time you restart Juli.

When Juli shuts down, the tunnel process is automatically terminated.

## Security

**Quick Tunnel mode is for personal development use only.** It is not a production deployment strategy.

When the tunnel is active, anyone with the URL can reach your Juli login page. Keep these in mind:

- Use a **strong, unique token** (`--token`). This is your only authentication layer.
- The tunnel URL is random and unlisted, but not secret. Treat it like a password-protected link.
- Cloudflare Quick Tunnels have no SLA or uptime guarantee.
- Juli uses token-based authentication. There is no rate limiting or multi-factor authentication.

For production or team use, consider a proper reverse proxy with TLS, or a mesh VPN like Tailscale.

## Troubleshooting

**`cloudflared not found`** — Install cloudflared (see Prerequisites above). Juli will continue serving locally without the tunnel.

**Tunnel fails to start** — If you have an existing `~/.cloudflared/config.yaml`, Quick Tunnels may conflict with it. Try renaming or removing the config file.

**Tunnel URL never appears** — Juli waits up to 15 seconds for cloudflared to produce a URL. Check your internet connection. If cloudflared exits with an error, Juli logs it and continues locally.

**Tunnel drops after a while** — Quick Tunnels are ephemeral. If cloudflared crashes, Juli logs a warning and continues serving locally. Restart Juli to get a new tunnel.

## Other Options

Juli works with any tool that can forward HTTP traffic to a local port. If you prefer a different solution:

- **ngrok:** `ngrok http 8080`
- **Tailscale Funnel:** `tailscale funnel 8080`
- **Reverse proxy:** Configure nginx or Caddy to proxy to `localhost:8080` with your own TLS certificate
- **SSH tunnel:** `ssh -R 80:localhost:8080 your-server`

Juli's `--tls-cert` and `--tls-key` flags provide native TLS if you bring your own certificates.
