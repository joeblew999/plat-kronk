package codeccheck

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/joeblew999/plat-kronk/internal/codecinstaller"
)

// Run executes the codeccheck CLI logic. It returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("codeccheck", flag.ContinueOnError)
	fs.SetOutput(stderr)

	jsonOutput := fs.Bool("json", false, "Output as JSON")
	help := fs.Bool("help", false, "Show install instructions")
	install := fs.Bool("install", false, "Download and install pre-built codec libraries")
	libPath := fs.String("lib", "./lib/codecs", "Directory to install codec libraries")
	version := fs.String("version", "", "Version to install (default: latest)")
	repo := fs.String("repo", "", "GitHub repo hosting releases (default: built-in)")
	upgrade := fs.Bool("upgrade", false, "Upgrade existing installation")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *help {
		fmt.Fprintln(stdout, codecinstaller.InstallInstructions())
		return 0
	}

	if *install {
		if err := runInstall(stdout, stderr, *libPath, *version, *repo, *upgrade); err != nil {
			return 1
		}
		return 0
	}

	status := codecinstaller.CheckSystem()

	if *jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(status)
		return 0
	}

	printStatus(stdout, status)
	return 0
}

func runInstall(stdout, stderr io.Writer, libPath, version, repo string, upgrade bool) error {
	fmt.Fprintln(stdout, "╔══════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(stdout, "║         Installing Codec Libraries                           ║")
	fmt.Fprintln(stdout, "╚══════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(stdout)

	installed, info, _ := codecinstaller.CheckInstalled(libPath)
	if installed && !upgrade {
		fmt.Fprintf(stdout, "✅ Already installed at %s\n", libPath)
		fmt.Fprintf(stdout, "   Version: %s\n", info.Codecs["version"])
		fmt.Fprintf(stdout, "   Use -upgrade to force reinstall\n")
		return nil
	}

	cfg := codecinstaller.Config{
		LibPath:      libPath,
		Version:      version,
		ReleaseRepo:  repo,
		AllowUpgrade: upgrade,
	}

	fmt.Fprintf(stdout, "Installing to: %s\n", libPath)
	if version != "" {
		fmt.Fprintf(stdout, "Version: %s\n", version)
	} else {
		fmt.Fprintln(stdout, "Version: latest")
	}
	fmt.Fprintln(stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Fprintln(stdout, "Downloading...")
	if err := codecinstaller.Install(ctx, cfg); err != nil {
		fmt.Fprintf(stderr, "❌ Installation failed: %v\n", err)
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Fallback options:")
		fmt.Fprintln(stderr, "  1. Use openh264 (bundled, no install needed)")
		fmt.Fprintln(stderr, "  2. Manual install: go run ./cmd/codeccheck -help")
		return err
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "✅ Codec libraries installed successfully!")
	fmt.Fprintf(stdout, "   Location: %s\n", libPath)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "To use, set CGO flags:")
	fmt.Fprintf(stdout, "   export CGO_CFLAGS=\"-I%s/include\"\n", libPath)
	fmt.Fprintf(stdout, "   export CGO_LDFLAGS=\"-L%s/lib\"\n", libPath)
	return nil
}

func printStatus(stdout io.Writer, status codecinstaller.SystemStatus) {
	fmt.Fprintln(stdout, "╔══════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(stdout, "║         MediaDevices Codec Status                            ║")
	fmt.Fprintln(stdout, "╚══════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "System: %s/%s\n\n", status.OS, status.Arch)

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

		fmt.Fprintf(stdout, "%s %s%s\n", icon, codec.Name, detail)
	}

	fmt.Fprintln(stdout)

	if status.AllReady {
		fmt.Fprintln(stdout, "✅ All codecs available!")
	} else {
		fmt.Fprintln(stdout, "⚠️  Some codecs missing. Options:")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "1. RECOMMENDED: Use openh264 instead of x264")
		fmt.Fprintln(stdout, "   Change your import from:")
		fmt.Fprintln(stdout, "     \"github.com/pion/mediadevices/pkg/codec/x264\"")
		fmt.Fprintln(stdout, "   To:")
		fmt.Fprintln(stdout, "     \"github.com/pion/mediadevices/pkg/codec/openh264\"")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "2. Download pre-built codecs:")
		fmt.Fprintln(stdout, "   Run: go run ./cmd/codeccheck -install")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "3. Manual install:")
		fmt.Fprintln(stdout, "   Run: go run ./cmd/codeccheck -help")
	}
}
