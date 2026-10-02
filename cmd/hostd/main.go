package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/cloudberry/internal/eventlog"
	"github.com/nikbogman/cloudberry/internal/hostd"
)

func main() {
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

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("hostd listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
