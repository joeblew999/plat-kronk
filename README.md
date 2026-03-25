# plat-kronk

Multi-party video conferencing system using LiveKit + Cloudflare Tunnel.

## Prerequisites

Install xplat (cross-platform task runner with process-compose):

```sh
# Download from releases
https://github.com/joeblew999/xplat/releases/
```

## Quick Start

```sh
# Start the video conference (works immediately with dev defaults)
xplat task webrtc:up

# Open browser to the tunnel URL displayed in logs
# Or scan the QR code on mobile
```

## Commands

| Command | Description |
|---------|-------------|
| `xplat task webrtc:up` | Start all services (LiveKit, demo server, Caddy, tunnel) |
| `xplat task webrtc:attach` | Attach TUI to running services |
| `xplat task webrtc:down` | Stop all services |
| `xplat task env:init` | Generate `.env` with secure random secret |
| `xplat task env:show` | Show current environment config (secrets masked) |

## Configuration

### Development (Zero Config)

For local development, everything works out of the box with hardcoded defaults:

```sh
xplat task webrtc:up  # Just works!
```

### Production (Secure Secrets)

For production or when distributing to users, generate unique secrets:

```sh
# Generate .env with random 48-character secret
xplat task env:init

# Start with secure secret
xplat task webrtc:up
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LIVEKIT_API_KEY` | `devkey` | LiveKit API key |
| `LIVEKIT_API_SECRET` | (dev secret) | LiveKit API secret (min 32 chars) |
| `LIVEKIT_HOST` | `ws://localhost:7880` | LiveKit WebSocket URL |

See [.env.example](.env.example) for reference.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   Browser/App   │────▶│  Cloudflare     │
│                 │     │  Tunnel (HTTPS) │
└─────────────────┘     └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │     Caddy       │
                        │  (Reverse Proxy)│
                        └────────┬────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
     ┌────────▼────────┐ ┌───────▼───────┐ ┌───────▼───────┐
     │  WebRTC Demo    │ │    LiveKit    │ │   LiveKit     │
     │  (Token Gen)    │ │    Server     │ │   WebSocket   │
     │  :9080          │ │    :7880      │ │   /rtc        │
     └─────────────────┘ └───────────────┘ └───────────────┘
```

## Features

- Multi-party video conferencing
- Screen sharing
- Chat messaging
- Recording support
- QR code for mobile access
- Works on any device with camera (iOS, Android, desktop)

## Subsystems

| Subsystem | Description |
|-----------|-------------|
| `livekit` | WebRTC SFU server |
| `caddy` | Reverse proxy with automatic HTTPS |
| `cloudflared` | Cloudflare tunnel for public access |
| `webrtc-demo` | Demo web application |
