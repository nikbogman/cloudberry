package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/internal/compute"
	"github.com/nikbogman/homelab/internal/eventlog"
)

func main() {
	cfg, err := compute.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "compute-api")

	suspender := compute.NewSystemSuspender()

	handler, err := compute.NewHandler(cfg.UIOrigin, suspender, logger, cfg.Host)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("compute-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
