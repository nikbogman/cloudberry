package main

import (
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/control-plane/internal/eventlog"
	"github.com/nikbogman/homelab/control-plane/internal/gateway"
)

// .gitkeep satisfies go:embed's "at least one file" requirement before
// deploy_gateway.py builds the UI into uidist/.
//
//go:embed all:uidist
var embeddedUI embed.FS

func main() {
	uiAssets, err := fs.Sub(embeddedUI, "uidist")
	if err != nil {
		log.Fatal(err)
	}

	cfg := gateway.ConfigFromEnv()
	cfg.UIAssets = uiAssets

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
