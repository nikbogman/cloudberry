// Real entrypoint: wires piapi.NewHandler to environment-provided config so
// this can actually be run under systemd, not just imported for tests.
package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/nikbogman/homelab/services/internal/eventlog"
	"github.com/nikbogman/homelab/services/internal/piapi"
)

func main() {
	host := envOr("PI_API_HOST", "127.0.0.1")
	port := envOr("PI_API_PORT", "5000")

	logger := eventlog.NewGrafanaCloudLogger(
		mustEnv("GRAFANA_CLOUD_LOKI_URL"),
		mustEnv("GRAFANA_CLOUD_LOKI_USER"),
		mustEnv("GRAFANA_CLOUD_LOKI_API_KEY"),
		"pi-api",
	)

	handler, err := piapi.NewHandler(mustEnv("SERVER_MAC_ADDRESS"), piapi.NewWakeOnLanSender(), logger, host)
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("pi-api listening on %s", addr)
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
