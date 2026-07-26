package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/services/internal/envconfig"
	"github.com/nikbogman/homelab/services/internal/eventlog"
	"github.com/nikbogman/homelab/services/internal/gateway"
)

func main() {
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
		UIRoot:           "/srv/ui",
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
