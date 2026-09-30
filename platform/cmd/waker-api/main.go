package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/platform/internal/eventlog"
	"github.com/nikbogman/homelab/platform/internal/waker"
)

func main() {
	cfg := waker.ConfigFromEnv()

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "waker")

	handler, err := waker.NewHandler(cfg.Config, waker.NewWakeOnLanSender(), logger)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(cfg.BindHost, cfg.Port)
	log.Printf("waker listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
