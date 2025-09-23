package cmd

import (
	"os"
	logpkg "pipegen/internal/log"

	"github.com/spf13/viper"
)

// init runs after root flags are defined; configure global logger here.
func init() {
	// Environment variable override takes precedence over flag binding if explicitly set.
	envLevel := os.Getenv("PIPEGEN_LOG_LEVEL")
	levelStr := viper.GetString("log_level")
	if envLevel != "" {
		levelStr = envLevel
	}
	lvl, err := logpkg.ParseLevel(levelStr)
	logger := logpkg.NewSimple(lvl)
	if err != nil {
		// Fallback to info but record the issue at warn level using temporary logger
		logger.Warn("invalid log level requested, using info", "requested", levelStr, "error", err)
	}
	logpkg.SetGlobal(logger)
}
