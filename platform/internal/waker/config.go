package waker

import "github.com/nikbogman/homelab/platform/internal/env"

// EnvConfig is Config plus the waker's own listen port, both read from
// the process environment.
type EnvConfig struct {
	Config
	Port string
}

// ConfigFromEnv reads the Waker's runtime configuration. SLEEPER_MAC_ADDRESS
// has no safe default, so it's required.
func ConfigFromEnv() EnvConfig {
	return EnvConfig{
		Config: Config{
			MACAddress: env.MustEnv("SLEEPER_MAC_ADDRESS"),
			BindHost:   env.EnvOr("WAKER_HOST", "127.0.0.1"),
		},
		Port: env.EnvOr("WAKER_PORT", "5000"),
	}
}
