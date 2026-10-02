package edge

import (
	"log"
	"net/url"

	"github.com/nikbogman/homelab/internal/env"
)

// EnvConfig is Config plus Edge's listen port and tsnet state dir, all
// read from the process environment.
type EnvConfig struct {
	Config
	Port string
	// tsnet's node state. Must survive redeploys (a Railway volume),
	// or the node re-registers under a new name each time.
	TSStateDir string
}

// ConfigFromEnv reads Edge's runtime configuration. UIAssets is left
// unset -- callers fill it in from the embedded UI. PORT is set by Railway.
func ConfigFromEnv() EnvConfig {
	return EnvConfig{
		Config: Config{
			WakerAPIURL:   mustURL("WAKER_API_URL"),
			SleeperAPIURL: mustURL("SLEEPER_API_URL"),
			TailnetDomain: env.MustEnv("TAILNET_DOMAIN"),
		},
		Port:       env.EnvOr("PORT", "8080"),
		TSStateDir: env.MustEnv("TS_STATE_DIR"),
	}
}

func mustURL(name string) *url.URL {
	u, err := url.Parse(env.MustEnv(name))
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Fatalf("environment variable %s must be an absolute URL, got %q", name, u)
	}
	return u
}
