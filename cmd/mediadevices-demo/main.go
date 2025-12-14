// mediadevices-demo is a tiny smoke test that exercises the bundled OpenH264
// encoder using the mediadevices videotest driver (no real camera, no OS deps).
// It reads a single encoded chunk to prove the pipeline works without touching
// system-wide installs.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joeblew999/plat-kronk/pkg/mediadevicesdemo"
)

func main() {
	install := flag.Bool("install-codecs", false, "Download pre-built codec libraries before running demo")
	lib := flag.String("lib", "./lib/codecs", "Directory to install codec libraries (with -install-codecs)")
	version := flag.String("version", "", "Codec release version to install (default: latest)")
	repo := flag.String("repo", "", "Override GitHub repo for codec releases")
	upgrade := flag.Bool("upgrade", false, "Force reinstall codecs if already present")
	flag.Parse()

	cfg := mediadevicesdemo.Config{
		InstallCodecs: *install,
		LibPath:       *lib,
		Version:       *version,
		Repo:          *repo,
		Upgrade:       *upgrade,
	}

	if err := mediadevicesdemo.Run(os.Stdout, os.Stderr, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}
