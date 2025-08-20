package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

// SetVersionInfo sets the version information from build-time variables
func SetVersionInfo(v, c, bt string) {
	if v != "" {
		version = v
	}
	if c != "" {
		commit = c
	}
	if bt != "" {
		buildTime = bt
	}
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, build time, and commit information for PipeGen.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("PipeGen %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", buildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
