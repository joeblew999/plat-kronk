# WebRTC Video Conference Demo

Real-time video conferencing with local recording, built on livekit.

## Why This Matters

**Professional video apps record locally while streaming.** When you use Zoom, Riverside.fm, or Squadcast, they:

1. **Stream** a compressed, low-latency video to peers (WebRTC)
2. **Record** the full-quality source locally for post-production

This demo does both.

## Architecture

```
Camera (1080p) ──┬──► WebRTC Stream (compressed, real-time)
                 │    └─► sent to browser/peers
                 │
                 ├──► H264 Recording (video.h264 + audio.ogg)
                 │    └─► compressed, playable
                 │
                 └──► Raw Frames (PNG sequence)
                      └─► lossless, for editing
```

## Usage

```bash
# Basic: stream only
go run ./cmd/webrtc-demo

# With recording (compressed stream)
go run ./cmd/webrtc-demo -record ./recordings

# With raw frame capture (highest quality)
go run ./cmd/webrtc-demo -record ./recordings -record-raw
```

## Flags

| Flag | Description |
|------|-------------|
| `-record DIR` | Save compressed H264/Opus to directory |
| `-record-raw` | Also save raw PNG frames (large files) |
| `-no-open` | Don't auto-open browser |
| `-list` | List available cameras/mics |

## Output Files

```
recordings/
├── video.h264      # Compressed video stream
├── audio.ogg       # Opus audio
└── raw_frames/     # (if -record-raw)
    ├── frame_000001.png
    ├── frame_000002.png
    └── ...
```

## Post-Production

Convert raw frames to high-quality video:

```bash
ffmpeg -framerate 30 -i recordings/raw_frames/frame_%06d.png \
       -c:v libx264 -crf 18 -pix_fmt yuv420p output.mp4
```

Mux compressed recording:

```bash
ffmpeg -i recordings/video.h264 -i recordings/audio.ogg \
       -c:v copy -c:a copy output.mp4
```

## Tech Stack

- **pion/mediadevices** - camera/mic capture
- **pion/webrtc** - real-time streaming
- **OpenH264** - bundled encoder (no install required)
- **Opus** - audio codec
