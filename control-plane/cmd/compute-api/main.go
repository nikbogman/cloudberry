package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/control-plane/internal/compute"
	"github.com/nikbogman/homelab/control-plane/internal/env"
	"github.com/nikbogman/homelab/control-plane/internal/eventlog"
)

func main() {
	host := env.EnvOr("COMPUTE_API_HOST", "127.0.0.1")
	port := env.EnvOr("COMPUTE_API_PORT", "5000")

	logger := eventlog.NewGrafanaCloudLogger(
		env.MustEnv("GRAFANA_CLOUD_LOKI_URL"),
		env.MustEnv("GRAFANA_CLOUD_LOKI_USER"),
		env.MustEnv("GRAFANA_CLOUD_LOKI_API_KEY"),
		"compute-api",
	)

	handler, err := compute.NewHandler(env.MustEnv("UI_ORIGIN"), compute.NewSystemSuspender(), logger, host)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("compute-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
