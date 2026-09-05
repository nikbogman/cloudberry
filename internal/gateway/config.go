package gateway

import "github.com/nikbogman/homelab/internal/env"

// EnvConfig is Config plus the gateway's own listen port, both read from
// the process environment.
type EnvConfig struct {
	Config
	Port string
}

// ConfigFromEnv reads gateway runtime configuration from the process
// environment. GATEWAY_HOST/GATEWAY_PORT default to loopback/5000;
// COMPUTE_MAC_ADDRESS, COMPUTE_HOST, and COMPUTE_PROXY_PORT have no safe
// default -- there's no single correct target across every device this
// binary might run on -- so they're required. COMPUTE_API_URL defaults
// to "" (same-origin), which is what a local dev run wants; a real
// Deploy always sets it. UIAssets is left unset; callers fill it in from
// the embedded UI.
func ConfigFromEnv() EnvConfig {
	return EnvConfig{
		Config: Config{
			MACAddress:       env.MustEnv("COMPUTE_MAC_ADDRESS"),
			BindHost:         env.EnvOr("GATEWAY_HOST", "127.0.0.1"),
			ComputeHost:      env.MustEnv("COMPUTE_HOST"),
			ComputeProxyPort: env.MustEnvInt("COMPUTE_PROXY_PORT"),
			ComputeAPIURL:    env.EnvOr("COMPUTE_API_URL", ""),
		},
		Port: env.EnvOr("GATEWAY_PORT", "5000"),
	}
}
