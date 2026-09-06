package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Environment variables recognised by ruckus. Every one of these is optional
// and is only used to change a flag's *default* — an explicitly passed flag
// always wins.
const (
	EnvDBPath          = "RUCKUS_DB_PATH"
	EnvLogFormat       = "RUCKUS_LOG_FORMAT"
	EnvDefaultDuration = "RUCKUS_DEFAULT_DURATION"
	EnvDefaultInterval = "RUCKUS_DEFAULT_INTERVAL"
)

// LogFormatHuman is the RUCKUS_LOG_FORMAT value that selects console output.
const (
	LogFormatHuman = "human"
	LogFormatJSON  = "json"
)

// envString returns a trimmed environment value, or fallback when unset/empty.
// A leading "~" is expanded to the user's home directory, since values written
// in a .env file are not shell-expanded.
func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return expandHome(value)
}

// expandHome replaces a leading "~" with the user's home directory. The value
// is returned unchanged when the home directory cannot be determined.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// envDuration parses a Go duration from the environment. An unparseable value
// is reported rather than silently ignored, so a typo in RUCKUS_DEFAULT_DURATION
// does not quietly change how long experiments run.
func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %w", key, raw, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("invalid %s value %q: must be greater than 0", key, raw)
	}
	return parsed, nil
}

// envHumanOutput reports whether RUCKUS_LOG_FORMAT asks for console output.
func envHumanOutput() (bool, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(EnvLogFormat)))
	switch raw {
	case "":
		return false, nil
	case LogFormatHuman:
		return true, nil
	case LogFormatJSON:
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s value %q: expected %q or %q", EnvLogFormat, raw, LogFormatJSON, LogFormatHuman)
	}
}
