package sleeper

import (
	"github.com/nikbogman/homelab/waker-service/internal/env"
)

// EnvConfig holds sleeper-api runtime configuration read from the process
// environment.
type EnvConfig struct {
	Host     string
	Port     string
	UIOrigin string
}

// ConfigFromEnv reads sleeper-api runtime configuration from the process
// environment. SLEEPER_API_HOST/PORT default to their current production
// values; UI_ORIGIN names the one browser origin CORS should allow, so it
// has no safe default and is required.
func ConfigFromEnv() (EnvConfig, error) {
	return EnvConfig{
		Host:     env.EnvOr("SLEEPER_API_HOST", "127.0.0.1"),
		Port:     env.EnvOr("SLEEPER_API_PORT", "5000"),
		UIOrigin: env.MustEnv("UI_ORIGIN"),
	}, nil
}
