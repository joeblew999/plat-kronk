# WebRTC Demo

MUST use taskfile that calls process compose that calls task file !!!
MUST work on HTTPS locally and on local iOS device !!
MUST use cloudflare tunnel and MUST use CADDY

MUST run under HTTPS !! 

## Quick Start

```bash
task webrtc:up
```

## Architecture

```
Taskfile.yml (root)
  └── includes webrtc: ./cmd/webrtc-demo/Taskfile.yml
        └── task webrtc:up
              └── process-compose.yml
                    ├── webrtc-demo (task webrtc:demo) :9080
                    ├── caddy (task webrtc:caddy) :8080 -> :9080
                    └── tunnel (task webrtc:tunnel) -> Caddy :8080 -> QR code
```

Traffic flow: `iPhone -> Cloudflare HTTPS -> tunnel -> Caddy :8080 -> demo :9080`

## Tasks

| Task | Description |
|------|-------------|
| `task webrtc:up` | Start all services with process-compose (TUI) |
| `task webrtc:up-headless` | Start all services headless |
| `task webrtc:demo` | Run just the Go WebRTC server on :9080 |
| `task webrtc:caddy` | Run Caddy reverse proxy |
| `task webrtc:tunnel` | Run cloudflared tunnel with QR code |
| `task webrtc:down` | Stop all processes |
| `task webrtc:clean` | Clean recordings |

## Ports

| Port | Service |
|------|---------|
| 9080 | WebRTC demo server |
| 8080 | Caddy reverse proxy |
| *.trycloudflare.com | Tunnel (public HTTPS) |
