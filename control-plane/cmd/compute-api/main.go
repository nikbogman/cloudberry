package main

import (
	"log"
	"net"
	"net/http"

	"github.com/nikbogman/homelab/control-plane/internal/compute"
	"github.com/nikbogman/homelab/control-plane/internal/envconfig"
	"github.com/nikbogman/homelab/control-plane/internal/eventlog"
)

func main() {
	host := envconfig.EnvOr("COMPUTE_API_HOST", "127.0.0.1")
	port := envconfig.EnvOr("COMPUTE_API_PORT", "5000")

	logger := eventlog.NewGrafanaCloudLogger(
		envconfig.MustEnv("GRAFANA_CLOUD_LOKI_URL"),
		envconfig.MustEnv("GRAFANA_CLOUD_LOKI_USER"),
		envconfig.MustEnv("GRAFANA_CLOUD_LOKI_API_KEY"),
		"compute-api",
	)

	// No env-driven override of the suspend command — a config knob here
	// could let a misconfiguration (e.g. "systemctl poweroff") reintroduce
	// a full shutdown path. NewSystemSuspender's default is the only
	// command this app will ever run.
	handler, err := compute.NewHandler(envconfig.MustEnv("UI_ORIGIN"), compute.NewSystemSuspender(), logger, host)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("compute-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
