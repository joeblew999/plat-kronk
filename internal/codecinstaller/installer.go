// Package codecinstaller provides Go-level idempotent installation of codec dependencies.
// This follows the pattern established by github.com/hybridgroup/yzma/pkg/download
// and github.com/ardanlabs/kronk/tools for llama.cpp libraries.
//
// For video conferencing with pion/mediadevices, codec libraries are needed:
//   - openh264: H.264 encoder (bundled as static libs - NO INSTALL NEEDED)
//   - x264: H.264 encoder (requires system install or pre-built libs)
//   - libvpx: VP8/VP9 encoder (requires system install or pre-built libs)
//   - opus: Audio encoder (requires system install or pre-built libs)
//
// The recommended approach is to use openh264 which is already bundled.
// This installer provides utilities for checking and managing codec availability.
//
// For runtime download of pre-built codec libraries, use the Install() function
// which downloads from GitHub releases (built by the build-codecs.yml workflow).
package codecinstaller

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const versionFile = "codecs-version.json"

// Errors
var (
	ErrUnsupportedOS   = errors.New("unsupported operating system")
	ErrCodecNotFound   = errors.New("codec not found")
	ErrInstallRequired = errors.New("codec installation required")
)

// CodecStatus represents the installation status of a codec.
type CodecStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Path      string `json:"path,omitempty"`
	Bundled   bool   `json:"bundled"` // true if static lib is bundled in package
}

// SystemStatus represents the overall codec availability.
type SystemStatus struct {
	OS       string        `json:"os"`
	Arch     string        `json:"arch"`
	Codecs   []CodecStatus `json:"codecs"`
	AllReady bool          `json:"all_ready"`
}

// CheckSystem verifies codec availability on the current system.
// It checks both bundled libraries (openh264) and system libraries (x264, vpx, opus).
func CheckSystem() SystemStatus {
	status := SystemStatus{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		Codecs: []CodecStatus{
			checkOpenH264(),
			checkX264(),
			checkVPX(),
			checkOpus(),
		},
	}

	status.AllReady = true
	for _, c := range status.Codecs {
		if !c.Available {
			status.AllReady = false
			break
		}
	}

	return status
}

// checkOpenH264 verifies openh264 is available (bundled in mediadevices).
func checkOpenH264() CodecStatus {
	// OpenH264 is bundled as static libraries in the mediadevices package.
	// No system installation required.
	return CodecStatus{
		Name:      "openh264",
		Available: true,
		Version:   "bundled",
		Bundled:   true,
	}
}

// checkX264 checks if x264 is available via pkg-config.
func checkX264() CodecStatus {
	return checkPkgConfig("x264", "x264")
}

// checkVPX checks if libvpx is available via pkg-config.
func checkVPX() CodecStatus {
	return checkPkgConfig("vpx", "libvpx")
}

// checkOpus checks if opus is available via pkg-config.
func checkOpus() CodecStatus {
	return checkPkgConfig("opus", "opus")
}

// checkPkgConfig uses pkg-config to check library availability.
func checkPkgConfig(pkgName, displayName string) CodecStatus {
	status := CodecStatus{
		Name:    displayName,
		Bundled: false,
	}

	// Check if pkg-config exists
	if _, err := exec.LookPath("pkg-config"); err != nil {
		return status
	}

	// Check if library exists
	cmd := exec.Command("pkg-config", "--exists", pkgName)
	if err := cmd.Run(); err != nil {
		return status
	}

	status.Available = true

	// Get version
	cmd = exec.Command("pkg-config", "--modversion", pkgName)
	if out, err := cmd.Output(); err == nil {
		status.Version = string(out[:len(out)-1]) // trim newline
	}

	return status
}

// InstallInstructions returns platform-specific installation instructions.
func InstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return `macOS Installation (Homebrew):
  brew install x264 libvpx opus pkg-config

Or use the zero-dependency option:
  Import openh264 instead of x264 (bundled, no install needed)`

	case "linux":
		return `Linux Installation:

Debian/Ubuntu:
  sudo apt install libx264-dev libvpx-dev libopus-dev pkg-config

Fedora/RHEL:
  sudo dnf install x264-devel libvpx-devel opus-devel pkgconfig

Arch Linux:
  sudo pacman -S x264 libvpx opus pkg-config

Alpine:
  sudo apk add x264-dev libvpx-dev opus-dev pkgconfig

Or use the zero-dependency option:
  Import openh264 instead of x264 (bundled, no install needed)`

	case "windows":
		return `Windows Installation:

Option 1 - vcpkg:
  vcpkg install x264:x64-windows libvpx:x64-windows opus:x64-windows

Option 2 - MSYS2:
  pacman -S mingw-w64-x86_64-x264 mingw-w64-x86_64-libvpx mingw-w64-x86_64-opus

Recommended: Use openh264 instead (bundled, no install needed)`

	default:
		return "Unknown OS. Please install x264, libvpx, and opus development libraries."
	}
}

// VersionInfo stores installed codec versions for idempotent checks.
type VersionInfo struct {
	Codecs map[string]string `json:"codecs"`
}

// SaveVersionInfo saves version information to disk for idempotent checks.
func SaveVersionInfo(dir string, info VersionInfo) error {
	path := filepath.Join(dir, versionFile)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal version info: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadVersionInfo loads version information from disk.
func LoadVersionInfo(dir string) (VersionInfo, error) {
	path := filepath.Join(dir, versionFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("read version info: %w", err)
	}

	var info VersionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return VersionInfo{}, fmt.Errorf("parse version info: %w", err)
	}

	return info, nil
}

// RecommendedCodec returns the recommended video codec based on system availability.
// It prefers openh264 (bundled) over x264 (requires install).
func RecommendedCodec() string {
	// OpenH264 is always available since it's bundled
	return "openh264"
}

// =============================================================================
// Runtime Download - Following yzma/kronk pattern
// =============================================================================

const (
	// DefaultReleaseRepo is the GitHub repo hosting pre-built codec libraries.
	// Built by .github/workflows/build-codecs.yml and released with "codecs-" prefix.
	DefaultReleaseRepo = "joeblew999/plat-kronk"

	// RetryCount is how many times to retry downloads.
	RetryCount = 3

	// RetryDelay is the delay between retries.
	RetryDelay = 2 * time.Second
)

// Config holds the configuration for codec installation.
type Config struct {
	// LibPath is where codec libraries will be installed.
	LibPath string

	// Version is the release version to download (e.g., "v1.0.0").
	// If empty, fetches the latest release.
	Version string

	// ReleaseRepo is the GitHub repo hosting releases (e.g., "org/repo").
	ReleaseRepo string

	// AllowUpgrade permits upgrading existing installations.
	AllowUpgrade bool
}

// Install downloads and installs pre-built codec libraries.
// This is idempotent - it checks version.json and skips if already installed.
//
// Example:
//
//	cfg := codecinstaller.Config{
//	    LibPath: "./lib",
//	    Version: "v1.0.0",  // or "" for latest
//	}
//	if err := codecinstaller.Install(context.Background(), cfg); err != nil {
//	    log.Fatal(err)
//	}
func Install(ctx context.Context, cfg Config) error {
	if cfg.ReleaseRepo == "" {
		cfg.ReleaseRepo = DefaultReleaseRepo
	}

	var existing *VersionInfo
	if info, err := LoadVersionInfo(cfg.LibPath); err == nil {
		existing = &info
		if !cfg.AllowUpgrade {
			if cfg.Version == "" || info.Codecs["version"] == cfg.Version {
				return nil // already installed and no upgrade requested
			}
		}
	}

	// Check if already installed
	version := cfg.Version
	if version == "" {
		var err error
		version, err = getLatestRelease(ctx, cfg.ReleaseRepo)
		if err != nil {
			return fmt.Errorf("get latest release: %w", err)
		}
	}

	if existing != nil && existing.Codecs["version"] == version {
		return nil // already at desired version
	}

	// Download and extract
	if err := downloadCodecs(ctx, cfg.ReleaseRepo, version, cfg.LibPath); err != nil {
		return fmt.Errorf("download codecs: %w", err)
	}

	// Save version info
	info := VersionInfo{
		Codecs: map[string]string{
			"version": version,
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
		},
	}
	if err := SaveVersionInfo(cfg.LibPath, info); err != nil {
		return fmt.Errorf("save version info: %w", err)
	}

	return nil
}

// CodecsReleasePrefix is the prefix for codec release tags.
const CodecsReleasePrefix = "codecs-"

// getLatestRelease fetches the latest codec release tag from GitHub.
// It looks for releases with the "codecs-" prefix.
func getLatestRelease(ctx context.Context, repo string) (string, error) {
	// List releases and find the latest one with codecs- prefix
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)

	var version string
	var lastErr error

	for i := 0; i < RetryCount; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/vnd.github+json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(RetryDelay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
			time.Sleep(RetryDelay)
			continue
		}

		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			lastErr = err
			time.Sleep(RetryDelay)
			continue
		}

		// Find the first release with codecs- prefix (releases are sorted by date desc)
		for _, r := range releases {
			if strings.HasPrefix(r.TagName, CodecsReleasePrefix) {
				version = r.TagName
				break
			}
		}
		break
	}

	if version == "" {
		if lastErr != nil {
			return "", fmt.Errorf("failed to get codec release after %d retries: %w", RetryCount, lastErr)
		}
		return "", fmt.Errorf("no codec releases found (looking for tags with '%s' prefix)", CodecsReleasePrefix)
	}

	return version, nil
}

// downloadCodecs downloads and extracts codec libraries.
func downloadCodecs(ctx context.Context, repo, version, destDir string) error {
	// Construct download URL
	filename := fmt.Sprintf("codecs-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, version, filename)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	// Download
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Extract tar.gz
	return extractTarGz(resp.Body, destDir)
}

// extractTarGz extracts a tar.gz archive to the destination directory.
func extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		// Security: prevent path traversal
		name := header.Name
		if strings.Contains(name, "..") {
			continue
		}

		target := filepath.Join(destDir, filepath.Clean(name))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("create parent dir: %w", err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("write file: %w", err)
			}
			f.Close()

		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil && !os.IsExist(err) {
				return fmt.Errorf("create symlink: %w", err)
			}
		}
	}

	return nil
}

// CheckInstalled checks if codecs are already installed at the given path.
func CheckInstalled(libPath string) (bool, VersionInfo, error) {
	info, err := LoadVersionInfo(libPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, VersionInfo{}, nil
		}
		return false, VersionInfo{}, err
	}
	return true, info, nil
}
