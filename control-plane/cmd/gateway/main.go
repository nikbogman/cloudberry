package main

import (
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/control-plane/internal/env"
	"github.com/nikbogman/homelab/control-plane/internal/eventlog"
	"github.com/nikbogman/homelab/control-plane/internal/gateway"
)

// uidist is populated by `npm run build` in ../../ui; deploy_gateway.py
// always runs that build first. The checked-in .gitkeep exists only so a
// fresh checkout compiles before the UI's been built -- go:embed requires
// at least one matching file.
//
//go:embed all:uidist
var embeddedUI embed.FS

func main() {
	uiAssets, err := fs.Sub(embeddedUI, "uidist")
	if err != nil {
		log.Fatal(err)
	}

	host := env.EnvOr("GATEWAY_HOST", "127.0.0.1")
	port := env.EnvOr("GATEWAY_PORT", "5000")

	logger := eventlog.NewGrafanaCloudLogger(
		env.MustEnv("GRAFANA_CLOUD_LOKI_URL"),
		env.MustEnv("GRAFANA_CLOUD_LOKI_USER"),
		env.MustEnv("GRAFANA_CLOUD_LOKI_API_KEY"),
		"gateway",
	)

	cfg := gateway.Config{
		MACAddress:       env.MustEnv("COMPUTE_MAC_ADDRESS"),
		BindHost:         host,
		UIAssets:         uiAssets,
		ComputeHost:      env.MustEnv("COMPUTE_HOST"),
		ComputeProxyPort: env.MustEnvInt("COMPUTE_PROXY_PORT"),
	}

	handler, err := gateway.NewHandler(cfg, gateway.NewWakeOnLanSender(), logger)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
