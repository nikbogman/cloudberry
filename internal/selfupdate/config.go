package selfupdate

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/nikbogman/metal/internal/env"
)

// Config is shared by gateway and hostd: how often to check for updates,
// and where releases come from (empty: binary updates off).
type Config struct {
	Every      time.Duration
	ReleaseURL string
}

// ConfigFromEnv reads UPDATE_MINUTES (default 2) and RELEASE_URL.
func ConfigFromEnv() Config {
	minutes, err := strconv.Atoi(env.EnvOr("UPDATE_MINUTES", "2"))
	if err != nil || minutes <= 0 {
		log.Fatalf("UPDATE_MINUTES must be a positive integer")
	}
	return Config{
		Every:      time.Duration(minutes) * time.Minute,
		ReleaseURL: env.EnvOr("RELEASE_URL", "https://github.com/nikbogman/metal"),
	}
}

// Updater for the running binary bin, or nil if RELEASE_URL is empty. Release
// assets are named <bin>-linux-<GOARCH>.
func (c Config) Updater(bin, version string) *Updater {
	if c.ReleaseURL == "" {
		return nil
	}
	path, err := os.Executable()
	if err != nil {
		path = "/usr/local/bin/" + bin
	}
	return &Updater{BaseURL: c.ReleaseURL, Asset: bin + "-linux-" + runtime.GOARCH, BinPath: path, Version: version}
}
