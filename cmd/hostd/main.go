package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/nikbogman/metal/internal/cors"
	"github.com/nikbogman/metal/internal/eventlog"
	"github.com/nikbogman/metal/internal/hostd"
	"github.com/nikbogman/metal/internal/selfupdate"
)

// version is set at build time (-ldflags "-X main.version=..."): a release
// tag for release builds, which update themselves; anything else doesn't.
var version = "dev"

func main() {
	log.Printf("hostd %s", version)
	cfg, err := hostd.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "hostd")

	suspender := hostd.NewSystemSuspender()

	handler, err := hostd.NewHandler(suspender, logger, cfg.Host)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	var stacks *hostd.StackSync
	if cfg.StacksDir != "" {
		stacks = hostd.NewStackSync(cfg.StacksDir, cfg.FirewallDropIn, logger)
		mux.Handle("GET /routes", stacks.RoutesHandler())
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("hostd listening on %s", addr)
	go func() { log.Fatal(http.ListenAndServe(addr, cors.Allow(cfg.UIOrigin, mux))) }()

	// One loop, so a sync and a binary swap never overlap. Returning exits
	// hostd; systemd (Restart=always) starts the new binary.
	updater := cfg.Update.Updater("hostd", version)
	reportSync, reportUpdate := reporter(logger, "stacks_sync_failed"), reporter(logger, "update_failed")
	ctx := context.Background()
	for {
		if stacks != nil {
			reportSync(stacks.Sync(ctx))
		}
		if updater != nil {
			tag, updated, err := updater.Update(ctx)
			reportUpdate(err)
			if updated {
				logger.SendEvent("binary_updated", "succeeded", nil, map[string]any{"from": version, "to": tag})
				log.Print("restarting for an update")
				return
			}
		}
		time.Sleep(cfg.Update.Every)
	}
}

// reporter logs each distinct error once as eventType, not every poll: a
// GitHub outage would otherwise log every few minutes. nil resets it.
func reporter(logger *eventlog.GrafanaCloudLogger, eventType string) func(error) {
	var last string
	return func(err error) {
		if err == nil {
			last = ""
			return
		}
		if err.Error() == last {
			return
		}
		last = err.Error()
		if errors.Is(err, selfupdate.ErrLocalBuild) {
			log.Printf("self-update off: %v", err)
			return
		}
		logger.SendEvent(eventType, "failed", nil, map[string]any{"error": err.Error()})
	}
}
