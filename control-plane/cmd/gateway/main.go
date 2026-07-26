package main

import (
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/control-plane/internal/envconfig"
	"github.com/nikbogman/homelab/control-plane/internal/eventlog"
	"github.com/nikbogman/homelab/control-plane/internal/gateway"
)

// uidist is populated by `npm run build` in ../../ui (vite.config.ts's
// outDir) before this package is compiled -- provisioning/deploy_gateway.py
// always builds the UI first. The checked-in .gitkeep placeholder is only
// there so a fresh checkout compiles before the UI's ever been built;
// go:embed requires the pattern to match at least one file.
//
//go:embed all:uidist
var embeddedUI embed.FS

func main() {
	uiAssets, err := fs.Sub(embeddedUI, "uidist")
	if err != nil {
		log.Fatal(err)
	}

	host := envconfig.EnvOr("GATEWAY_HOST", "127.0.0.1")
	port := envconfig.EnvOr("GATEWAY_PORT", "5000")

	logger := eventlog.NewGrafanaCloudLogger(
		envconfig.MustEnv("GRAFANA_CLOUD_LOKI_URL"),
		envconfig.MustEnv("GRAFANA_CLOUD_LOKI_USER"),
		envconfig.MustEnv("GRAFANA_CLOUD_LOKI_API_KEY"),
		"gateway",
	)

	cfg := gateway.Config{
		MACAddress:       envconfig.MustEnv("COMPUTE_MAC_ADDRESS"),
		BindHost:         host,
		UIAssets:         uiAssets,
		ComputeHost:      envconfig.MustEnv("COMPUTE_HOST"),
		ComputeProxyPort: envconfig.MustEnvInt("COMPUTE_PROXY_PORT"),
	}

	handler, err := gateway.NewHandler(cfg, gateway.NewWakeOnLanSender(), logger)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
