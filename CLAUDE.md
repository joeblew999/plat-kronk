# Claude Assistant Notes



### Codec Libraries (for pion/mediadevices)

Pre-built static libraries for video conferencing with pion/mediadevices.

**CLI:** `cmd/codeccheck/main.go`
**Taskfile:** `taskfiles/Taskfile.codecs.yml`
**Workflow:** `.github/workflows/build-codecs.yml`

**Commands:**
```bash
task codecs:status             # Check codec availability
task codecs:install            # Download pre-built libraries
task codecs:recommend          # Show zero-dependency option
task codecs:build:trigger      # Trigger build workflow (maintainers)
```

**User Experience:**
1. **RECOMMENDED** (zero install): Use openh264 instead of x264
   ```go
   // Change from: "github.com/pion/mediadevices/pkg/codec/x264"
   // To:          "github.com/pion/mediadevices/pkg/codec/openh264"
   ```
2. **Pre-built download**: `task codecs:install` downloads from GitHub releases
3. **Manual install**: `brew install x264 libvpx opus` (macOS)

**Build Workflow:**
GitHub Actions builds x264, libvpx, opus as static libraries for:
- darwin-arm64, darwin-amd64
- linux-amd64, linux-arm64
- windows-amd64

Releases are tagged `codecs-v1.0.0` and users download via `codeccheck -install`.






