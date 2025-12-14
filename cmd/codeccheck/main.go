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
	"os"

	codeccheckcli "github.com/joeblew999/plat-kronk/pkg/codeccheck"
)

func main() {
	code := codeccheckcli.Run(os.Args[1:], os.Stdout, os.Stderr)
	if code != 0 {
		os.Exit(code)
	}
}
