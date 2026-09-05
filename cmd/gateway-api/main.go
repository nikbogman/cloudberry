package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/internal/eventlog"
	"github.com/nikbogman/homelab/internal/gateway"
	"github.com/nikbogman/homelab/ui"
)

func main() {
	cfg := gateway.ConfigFromEnv()
	cfg.UIAssets = ui.Assets

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "gateway")

	handler, err := gateway.NewHandler(cfg.Config, gateway.NewWakeOnLanSender(), logger)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(cfg.BindHost, cfg.Port)
	log.Printf("gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
