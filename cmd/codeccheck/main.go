// codeccheck - Check and install codec dependencies
//
// This tool checks if the required codec libraries are available on the system
// and can download pre-built libraries from GitHub releases.
// Following the kronk/yzma pattern of idempotent Go-level dependency management.
//
// Usage:
//
//	go run ./cmd/codeccheck              # Check status
//	go run ./cmd/codeccheck -install     # Download pre-built codecs
//	go run ./cmd/codeccheck -help        # Show manual install instructions
//
// The recommended approach is to use openh264 which requires NO system installation.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joeblew999/plat-kronk/internal/codecinstaller"
)

func main() {
	jsonOutput := flag.Bool("json", false, "Output as JSON")
	help := flag.Bool("help", false, "Show install instructions")
	install := flag.Bool("install", false, "Download and install pre-built codec libraries")
	libPath := flag.String("lib", "./lib/codecs", "Directory to install codec libraries")
	version := flag.String("version", "", "Version to install (default: latest)")
	repo := flag.String("repo", "", "GitHub repo hosting releases (default: built-in)")
	upgrade := flag.Bool("upgrade", false, "Upgrade existing installation")
	flag.Parse()

	if *help {
		fmt.Println(codecinstaller.InstallInstructions())
		return
	}

	// Handle install command
	if *install {
		runInstall(*libPath, *version, *repo, *upgrade)
		return
	}

	status := codecinstaller.CheckSystem()

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(status)
		return
	}

	// Human-readable output
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         MediaDevices Codec Status                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("System: %s/%s\n\n", status.OS, status.Arch)

	for _, codec := range status.Codecs {
		icon := "❌"
		if codec.Available {
			icon = "✅"
		}

		detail := ""
		if codec.Bundled {
			detail = " (bundled - no install needed)"
		} else if codec.Version != "" {
			detail = fmt.Sprintf(" v%s", codec.Version)
		} else {
			detail = " (not installed)"
		}

		fmt.Printf("%s %s%s\n", icon, codec.Name, detail)
	}

	fmt.Println()

	if status.AllReady {
		fmt.Println("✅ All codecs available!")
	} else {
		fmt.Println("⚠️  Some codecs missing. Options:")
		fmt.Println()
		fmt.Println("1. RECOMMENDED: Use openh264 instead of x264")
		fmt.Println("   Change your import from:")
		fmt.Println("     \"github.com/pion/mediadevices/pkg/codec/x264\"")
		fmt.Println("   To:")
		fmt.Println("     \"github.com/pion/mediadevices/pkg/codec/openh264\"")
		fmt.Println()
		fmt.Println("2. Download pre-built codecs:")
		fmt.Println("   Run: go run ./cmd/codeccheck -install")
		fmt.Println()
		fmt.Println("3. Manual install:")
		fmt.Println("   Run: go run ./cmd/codeccheck -help")
	}
}

// runInstall downloads and installs pre-built codec libraries.
func runInstall(libPath, version, repo string, upgrade bool) {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         Installing Codec Libraries                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Check if already installed
	installed, info, _ := codecinstaller.CheckInstalled(libPath)
	if installed && !upgrade {
		fmt.Printf("✅ Already installed at %s\n", libPath)
		fmt.Printf("   Version: %s\n", info.Codecs["version"])
		fmt.Printf("   Use -upgrade to force reinstall\n")
		return
	}

	cfg := codecinstaller.Config{
		LibPath:      libPath,
		Version:      version,
		ReleaseRepo:  repo,
		AllowUpgrade: upgrade,
	}

	fmt.Printf("Installing to: %s\n", libPath)
	if version != "" {
		fmt.Printf("Version: %s\n", version)
	} else {
		fmt.Println("Version: latest")
	}
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("Downloading...")
	if err := codecinstaller.Install(ctx, cfg); err != nil {
		fmt.Printf("❌ Installation failed: %v\n", err)
		fmt.Println()
		fmt.Println("Fallback options:")
		fmt.Println("  1. Use openh264 (bundled, no install needed)")
		fmt.Println("  2. Manual install: go run ./cmd/codeccheck -help")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ Codec libraries installed successfully!")
	fmt.Printf("   Location: %s\n", libPath)
	fmt.Println()
	fmt.Println("To use, set CGO flags:")
	fmt.Printf("   export CGO_CFLAGS=\"-I%s/include\"\n", libPath)
	fmt.Printf("   export CGO_LDFLAGS=\"-L%s/lib\"\n", libPath)
}
