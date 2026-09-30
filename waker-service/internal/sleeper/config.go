package sleeper

import (
	"github.com/nikbogman/homelab/waker-service/internal/env"
)

// EnvConfig holds sleeper-api runtime configuration read from the process
// environment.
type EnvConfig struct {
	Host string
	Port string
}

// ConfigFromEnv reads sleeper-api runtime configuration from the process
// environment. SLEEPER_API_HOST/PORT default to their current production
// values.
func ConfigFromEnv() (EnvConfig, error) {
	return EnvConfig{
		Host: env.EnvOr("SLEEPER_API_HOST", "127.0.0.1"),
		Port: env.EnvOr("SLEEPER_API_PORT", "5000"),
	}, nil
}
