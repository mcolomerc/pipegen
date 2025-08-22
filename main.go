package main

import (
	"embed"
	"fmt"
	"os"

	"pipegen/cmd"
	"pipegen/internal/dashboard"
)

//go:embed web
var webFiles embed.FS

// Build-time variables
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	// Set version info in cmd package
	cmd.SetVersionInfo(version, commit, buildTime)

	// Initialize web files for dashboard package
	dashboard.SetWebFiles(webFiles)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
