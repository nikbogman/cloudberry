package waker

import "github.com/nikbogman/homelab/internal/env"

// EnvConfig is Config plus the waker's own listen port, both read from
// the process environment.
type EnvConfig struct {
	Config
	Port string
}

// ConfigFromEnv reads waker runtime configuration from the process
// environment. WAKER_HOST/WAKER_PORT default to loopback/5000;
// COMPUTE_MAC_ADDRESS has no safe default, so it's required.
// COMPUTE_API_URL defaults
// to "" (same-origin), which is what a local dev run wants; a real
// Deploy always sets it. UIAssets is left unset; callers fill it in from
// the embedded UI.
func ConfigFromEnv() EnvConfig {
	return EnvConfig{
		Config: Config{
			MACAddress:    env.MustEnv("COMPUTE_MAC_ADDRESS"),
			BindHost:      env.EnvOr("WAKER_HOST", "127.0.0.1"),
			ComputeAPIURL: env.EnvOr("COMPUTE_API_URL", ""),
		},
		Port: env.EnvOr("WAKER_PORT", "5000"),
	}
}
