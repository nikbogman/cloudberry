package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/internal/eventlog"
	"github.com/nikbogman/homelab/internal/sleeper"
)

func main() {
	cfg, err := sleeper.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "sleeper-api")

	suspender := sleeper.NewSystemSuspender()

	handler, err := sleeper.NewHandler(suspender, logger, cfg.Host)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("sleeper-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
