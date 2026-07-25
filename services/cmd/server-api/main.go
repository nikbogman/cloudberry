// Real entrypoint: wires serverapi.NewHandler to environment-provided
// config so this can actually be run under systemd, not just imported for
// tests.
package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/nikbogman/homelab/services/internal/eventlog"
	"github.com/nikbogman/homelab/services/internal/serverapi"
)

func main() {
	host := envOr("SERVER_API_HOST", "127.0.0.1")
	port := envOr("SERVER_API_PORT", "5000")

	logger := eventlog.NewGrafanaCloudLogger(
		mustEnv("GRAFANA_CLOUD_LOKI_URL"),
		mustEnv("GRAFANA_CLOUD_LOKI_USER"),
		mustEnv("GRAFANA_CLOUD_LOKI_API_KEY"),
		"server-api",
	)

	// No env-driven override of the suspend command — a config knob here
	// could let a misconfiguration (e.g. "systemctl poweroff") reintroduce
	// a full shutdown path. NewSystemSuspender's default is the only
	// command this app will ever run.
	handler, err := serverapi.NewHandler(mustEnv("UI_ORIGIN"), serverapi.NewSystemSuspender(), logger, host)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("server-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func mustEnv(name string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		log.Fatalf("missing required environment variable %s", name)
	}
	return value
}

func envOr(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}
