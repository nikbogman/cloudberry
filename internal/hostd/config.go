package hostd

import (
	"github.com/nikbogman/cloudberry/internal/env"
)

// EnvConfig holds hostd runtime configuration read from the process
// environment.
type EnvConfig struct {
	Host     string
	Port     string
	UIOrigin string
}

// ConfigFromEnv reads hostd runtime configuration from the process
// environment. HOSTD_HOST/PORT default to their current production
// values. UI_ORIGIN is the
// UI's Cloudflare Pages origin, allowed by CORS.
func ConfigFromEnv() (EnvConfig, error) {
	return EnvConfig{
		Host:     env.EnvOr("HOSTD_HOST", "127.0.0.1"),
		Port:     env.EnvOr("HOSTD_PORT", "5000"),
		UIOrigin: env.EnvOr("UI_ORIGIN", ""),
	}, nil
}
