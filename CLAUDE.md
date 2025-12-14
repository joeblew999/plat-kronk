# Claude Assistant Notes



### Codec Libraries (for pion/mediadevices)

Pre-built static libraries for video conferencing with pion/mediadevices.

**CLI:** `cmd/codeccheck/main.go`
**Taskfile:** `taskfiles/Taskfile.codecs.yml`
**Workflow:** `.github/workflows/build-codecs.yml`
  - Builds x264/libvpx/opus static libs for macOS, Linux, Windows (all architectures in the matrix)
  - Publishes `codecs-vX.Y.Z` releases so users never build locally; they just download via `codeccheck -install`

**Commands:**
```bash
task codecs:status             # Check codec availability
task codecs:install            # Download pre-built libraries
task codecs:recommend          # Show zero-dependency option
task codecs:build:trigger      # Trigger build workflow (maintainers)
task mediadevices:demo         # Demo without download (bundled openh264)
task mediadevices:demo:install # Demo with codec download/upgrade to lib/codecs (preferred at runtime)
```

**Source Mirror:**
- `.src/mediadevices` holds a git clone of https://github.com/pion/mediadevices for local reference.

**Local-Only Rule:**
- We MUST NOT pollute the OS; keep builds and experiments contained (use `.src` and project-local dirs, prefer bundled openh264 or prebuilt libs).

**Platform Support:**
- Everything must work on macOS, Linux, and Windows. Demo uses `videotest` + bundled OpenH264 (no hardware or system drivers).
- Do not assume Task or GH CLI are installed—always provide plain `go run` fallbacks.

**Assumptions (minimal):**
- Go toolchain installed; outbound HTTPS to GitHub for fetching releases.
- No cameras or system codecs required; demo uses dummy driver.
- Project must not modify system-level packages or environment.

**User Experience:**
1. **RECOMMENDED** (zero install): Use openh264 instead of x264
   ```go
   // Change from: "github.com/pion/mediadevices/pkg/codec/x264"
   // To:          "github.com/pion/mediadevices/pkg/codec/openh264"
   ```
2. **Pre-built download**: `task codecs:install` downloads from GitHub releases
   - Latest release: `codecs-v0.1.0` (assets for darwin-arm64, linux-{amd64,arm64}, windows-amd64)
   - Runtime preference: mediadevices demo prepends `lib/codecs` to loader path (DYLD/LD/PATH) when present. `mediadevices:demo:install` runs with `-upgrade` to pull the latest tag; installer now skips if already on the target version.
   - If Task is not installed: `GOWORK=off go run ./cmd/codeccheck -install -lib ./lib/codecs`
3. **Smoke test**:
   - With Task: `task mediadevices:demo` or `task mediadevices:demo:install`
   - Without Task: `GOWORK=off go run ./cmd/mediadevices-demo [-install-codecs -upgrade -lib ./lib/codecs]`

**Build Workflow:**
GitHub Actions builds x264, libvpx, opus as static libraries for:
- darwin-arm64, darwin-amd64
- linux-amd64, linux-arm64
- windows-amd64

Releases are tagged `codecs-vX.Y.Z` (current: `codecs-v0.1.0`) and users download via `codeccheck -install`.
