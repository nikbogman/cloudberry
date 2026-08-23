package main

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

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

	cpuBaselinePercent, err := strconv.ParseFloat(env.EnvOr("COMPUTE_API_CPU_BASELINE_PERCENT", strconv.FormatFloat(compute.DefaultCPUBaselinePercent, 'f', -1, 64)), 64)
	if err != nil {
		log.Fatalf("COMPUTE_API_CPU_BASELINE_PERCENT: %v", err)
	}
	runtime, err := compute.NewDockerRuntime(cpuBaselinePercent)
	if err != nil {
		log.Fatal(err)
	}
	suspender := compute.NewSystemSuspender()

	idleTimeout, err := time.ParseDuration(env.EnvOr("COMPUTE_API_IDLE_TIMEOUT", "1h"))
	if err != nil {
		log.Fatalf("COMPUTE_API_IDLE_TIMEOUT: %v", err)
	}
	dryRun, err := strconv.ParseBool(env.EnvOr("COMPUTE_API_AUTOSUSPEND_DRY_RUN", "false"))
	if err != nil {
		log.Fatalf("COMPUTE_API_AUTOSUSPEND_DRY_RUN: %v", err)
	}
	signals := compute.NewActivitySignals(logger, idleTimeout, dryRun)

	handler, err := compute.NewHandler(env.MustEnv("UI_ORIGIN"), suspender, runtime, logger, host, signals)
	if err != nil {
		log.Fatal(err)
	}

	watcher := compute.NewIdleWatcher(suspender, runtime, logger, signals)
	go watcher.Run()

	addr := net.JoinHostPort(host, port)
	log.Printf("compute-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
