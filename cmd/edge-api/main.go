package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"tailscale.com/tsnet"

	"github.com/nikbogman/homelab/internal/edge"
	"github.com/nikbogman/homelab/internal/eventlog"
	"github.com/nikbogman/homelab/ui"
)

func main() {
	cfg := edge.ConfigFromEnv()
	cfg.UIAssets = ui.Assets

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "edge")

	// Joins the tailnet as its own node, only to dial out. TS_AUTHKEY must
	// be user-owned (untagged): `tailscale serve` only injects the identity
	// header the Waker/Sleeper APIs require for user-owned nodes.
	ts := &tsnet.Server{Hostname: "edge", Dir: cfg.TSStateDir, AuthKey: os.Getenv("TS_AUTHKEY")}
	defer ts.Close()
	if _, err := ts.Up(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Public bind by design -- Edge is the one off-tailnet entry point.
	addr := net.JoinHostPort("", cfg.Port)
	log.Printf("edge listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, edge.WithAccessLog(edge.NewHandler(cfg.Config, ts.HTTPClient().Transport), logger)))
}
