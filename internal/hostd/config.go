package hostd

import (
	"github.com/nikbogman/homelab/internal/env"
)

// EnvConfig holds hostd runtime configuration read from the process
// environment.
type EnvConfig struct {
	Host string
	Port string
}

// ConfigFromEnv reads hostd runtime configuration from the process
// environment. HOSTD_HOST/PORT default to their current production
// values.
func ConfigFromEnv() (EnvConfig, error) {
	return EnvConfig{
		Host: env.EnvOr("HOSTD_HOST", "127.0.0.1"),
		Port: env.EnvOr("HOSTD_PORT", "5000"),
	}, nil
}
