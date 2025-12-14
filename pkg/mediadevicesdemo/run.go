package mediadevicesdemo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joeblew999/plat-kronk/internal/codecinstaller"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/openh264"
	_ "github.com/pion/mediadevices/pkg/driver/videotest"
	"github.com/pion/mediadevices/pkg/prop"
)

// Config controls demo behavior.
type Config struct {
	InstallCodecs bool
	LibPath       string
	Version       string
	Repo          string
	Upgrade       bool
}

// Run executes the mediadevices smoke test using the videotest driver and bundled OpenH264.
// It reads a small encoded chunk to confirm the pipeline works without system codecs or hardware.
func Run(stdout, stderr io.Writer, cfg Config) error {
	if cfg.LibPath == "" {
		cfg.LibPath = "./lib/codecs"
	}

	if cfg.InstallCodecs {
		if err := installCodecs(stdout, stderr, cfg); err != nil {
			return err
		}
	}
	if err := preferLocalCodecs(cfg.LibPath); err != nil {
		fmt.Fprintf(stderr, "warning: %v\n", err)
	}

	fmt.Fprintln(stdout, "▶ mediadevices demo: OpenH264 + videotest (no OS install)")

	params, err := openh264.NewParams()
	if err != nil {
		return fmt.Errorf("openh264 params: %w", err)
	}
	params.BitRate = 200_000

	videoID, err := pickVideoTestDevice()
	if err != nil {
		return fmt.Errorf("videotest device: %w", err)
	}

	selector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&params),
	)

	stream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.DeviceID = prop.String(videoID)
		},
		Codec: selector,
	})
	if err != nil {
		return fmt.Errorf("get user media: %w", err)
	}

	videoTrack := stream.GetVideoTracks()[0].(*mediadevices.VideoTrack)
	defer videoTrack.Close()

	reader, err := videoTrack.NewEncodedIOReader(params.RTPCodec().MimeType)
	if err != nil {
		return fmt.Errorf("encoded reader: %w", err)
	}
	defer reader.Close()

	type result struct {
		n   int
		err error
	}
	done := make(chan result, 1)
	buf := make([]byte, 4096)
	go func() {
		n, rerr := reader.Read(buf)
		done <- result{n: n, err: rerr}
	}()

	select {
	case res := <-done:
		if res.err != nil && res.err != io.EOF {
			return fmt.Errorf("read encoded chunk: %w", res.err)
		}
		fmt.Fprintf(stdout, "✅ Received %d bytes of OpenH264-encoded video (mime=%s)\n", res.n, params.RTPCodec().MimeType)
		fmt.Fprintln(stdout, "   Source: videotest driver (no hardware needed)")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout: no encoded data after 5s")
	}

	return nil
}

func preferLocalCodecs(libPath string) error {
	if libPath == "" {
		return nil
	}
	abs, err := filepath.Abs(libPath)
	if err == nil {
		libPath = abs
	}
	if info, err := os.Stat(libPath); err != nil || !info.IsDir() {
		return nil // nothing to prefer if the directory doesn't exist yet
	}

	switch runtime.GOOS {
	case "darwin":
		prependEnv("DYLD_LIBRARY_PATH", libPath)
		prependEnv("DYLD_FALLBACK_LIBRARY_PATH", libPath)
	case "linux":
		prependEnv("LD_LIBRARY_PATH", libPath)
	case "windows":
		prependEnv("PATH", libPath)
	default:
		return nil
	}

	return nil
}

func prependEnv(key, value string) {
	cur := os.Getenv(key)
	if cur == "" {
		_ = os.Setenv(key, value)
		return
	}
	_ = os.Setenv(key, value+string(os.PathListSeparator)+cur)
}

func pickVideoTestDevice() (string, error) {
	devs := mediadevices.EnumerateDevices()
	for _, d := range devs {
		if d.Kind == mediadevices.VideoInput && strings.EqualFold(d.Label, "VideoTest") {
			return d.DeviceID, nil
		}
	}
	for _, d := range devs {
		if d.Kind == mediadevices.VideoInput {
			return d.DeviceID, nil
		}
	}
	return "", fmt.Errorf("no video devices found (videotest driver should be registered)")
}

func installCodecs(stdout, stderr io.Writer, cfg Config) error {
	fmt.Fprintln(stdout, "Installing pre-built codecs for demo...")

	installed, info, _ := codecinstaller.CheckInstalled(cfg.LibPath)
	if installed && !cfg.Upgrade {
		fmt.Fprintf(stdout, "✅ Already installed at %s (version: %s)\n", cfg.LibPath, info.Codecs["version"])
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conf := codecinstaller.Config{
		LibPath:      cfg.LibPath,
		Version:      cfg.Version,
		ReleaseRepo:  cfg.Repo,
		AllowUpgrade: cfg.Upgrade,
	}

	fmt.Fprintf(stdout, "Installing to: %s\n", cfg.LibPath)
	if cfg.Version != "" {
		fmt.Fprintf(stdout, "Version: %s\n", cfg.Version)
	} else {
		fmt.Fprintln(stdout, "Version: latest")
	}
	fmt.Fprintln(stdout, "Downloading...")

	if err := codecinstaller.Install(ctx, conf); err != nil {
		fmt.Fprintf(stderr, "❌ Installation failed: %v\n", err)
		fmt.Fprintln(stderr, "Fallback: use bundled openh264 (no install needed).")
		return err
	}

	fmt.Fprintln(stdout, "✅ Codec libraries installed successfully!")
	fmt.Fprintf(stdout, "   Location: %s\n", cfg.LibPath)
	return nil
}
