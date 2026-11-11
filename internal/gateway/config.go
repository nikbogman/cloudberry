package gateway

import (
	"github.com/nikbogman/metal/internal/env"
	"github.com/nikbogman/metal/internal/selfupdate"
)

// EnvConfig is Config plus everything else the gateway reads from the
// process environment.
type EnvConfig struct {
	Config
	Port       string
	HostdURL   string
	RoutesFile string
	UIOrigin   string
	TsnetDir   string // tsnet node state, one subdirectory per node
	Hostname   string // gateway's own tsnet node: /wake, /settings, path routes
	StateFile  string // auto-wake/suspend switches and autoWoken, across restarts
	Update     selfupdate.Config
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
		TsnetDir:   env.EnvOr("GATEWAY_TSNET_DIR", "/var/lib/gateway/tsnet"),
		Hostname:   env.EnvOr("GATEWAY_HOSTNAME", "raspberry"),
		StateFile:  env.EnvOr("GATEWAY_STATE_FILE", "/var/lib/gateway/state.json"),
		Update:     selfupdate.ConfigFromEnv(),
	}
}
