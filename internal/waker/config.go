package waker

import "github.com/nikbogman/cloudberry/internal/env"

// EnvConfig is Config plus the waker's own listen port, both read from
// the process environment.
type EnvConfig struct {
	Config
	Port string
}

// ConfigFromEnv reads waker's runtime configuration. BLACKBERRY_MAC_ADDRESS
// has no safe default, so it's required.
func ConfigFromEnv() EnvConfig {
	return EnvConfig{
		Config: Config{
			MACAddress: env.MustEnv("BLACKBERRY_MAC_ADDRESS"),
			BindHost:   env.EnvOr("WAKER_HOST", "127.0.0.1"),
		},
		Port: env.EnvOr("WAKER_PORT", "5000"),
	}
}
