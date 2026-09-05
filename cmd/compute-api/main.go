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

	runtime, err := compute.NewDockerRuntime(cfg.CPUBaselinePercent)
	if err != nil {
		log.Fatal(err)
	}
	suspender := compute.NewSystemSuspender()

	signals := compute.NewActivitySignals(logger, cfg.IdleTimeout, cfg.AutosuspendDryRun)

	handler, err := compute.NewHandler(cfg.UIOrigin, suspender, runtime, logger, cfg.Host, signals)
	if err != nil {
		log.Fatal(err)
	}

	watcher := compute.NewIdleWatcher(suspender, runtime, logger, signals)
	go watcher.Run()

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("compute-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
