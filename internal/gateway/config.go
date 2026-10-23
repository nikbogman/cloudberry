package gateway

import "github.com/nikbogman/cloudberry/internal/env"

// EnvConfig is Config plus everything else the gateway reads from the
// process environment.
type EnvConfig struct {
	Config
	Port       string
	HostdURL   string
	RoutesFile string
	UIOrigin   string
}

// ConfigFromEnv reads the gateway's runtime configuration. BLACKBERRY_MAC_ADDRESS
// and HOSTD_URL have no safe default, so they're required.
func ConfigFromEnv() EnvConfig {
	return EnvConfig{
		Config: Config{
			MACAddress: env.MustEnv("BLACKBERRY_MAC_ADDRESS"),
			BindHost:   env.EnvOr("GATEWAY_HOST", "127.0.0.1"),
		},
		Port:       env.EnvOr("GATEWAY_PORT", "5000"),
		HostdURL:   env.MustEnv("HOSTD_URL"),
		RoutesFile: env.EnvOr("GATEWAY_ROUTES", "/etc/gateway/routes.json"),
		UIOrigin:   env.EnvOr("UI_ORIGIN", ""),
	}
}
