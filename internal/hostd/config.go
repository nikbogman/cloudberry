package hostd

import (
	"github.com/nikbogman/cloudberry/internal/env"
	"github.com/nikbogman/cloudberry/internal/selfupdate"
)

// EnvConfig holds hostd runtime configuration read from the process
// environment.
type EnvConfig struct {
	Host     string
	Port     string
	UIOrigin string
	// StacksDir is the deploys clone hostd syncs; empty turns the sync and
	// GET /routes off.
	StacksDir      string
	FirewallDropIn string
	Update         selfupdate.Config
}

// ConfigFromEnv reads hostd runtime configuration from the process
// environment. HOSTD_HOST/PORT default to their current production
// values. UI_ORIGIN is the
// UI's Cloudflare Pages origin, allowed by CORS.
func ConfigFromEnv() (EnvConfig, error) {
	return EnvConfig{
		Host:           env.EnvOr("HOSTD_HOST", "127.0.0.1"),
		Port:           env.EnvOr("HOSTD_PORT", "5000"),
		UIOrigin:       env.EnvOr("UI_ORIGIN", ""),
		StacksDir:      env.EnvOr("STACKS_DIR", ""),
		FirewallDropIn: env.EnvOr("FIREWALL_DROPIN", "/etc/systemd/system/stack-firewall.service.d/ports.conf"),
		Update:         selfupdate.ConfigFromEnv(),
	}, nil
}
