package eventlog

import "github.com/nikbogman/homelab/internal/env"

// GrafanaConfig holds the Grafana Cloud Loki credentials, read from the
// process environment identically by every entrypoint that logs via
// NewGrafanaCloudLogger.
type GrafanaConfig struct {
	LokiURL    string
	LokiUser   string
	LokiAPIKey string
}

// MustGrafanaConfig reads GrafanaConfig from the process environment,
// exiting the process if any field is unset -- there's no safe default
// for where events ship to.
func MustGrafanaConfig() GrafanaConfig {
	return GrafanaConfig{
		LokiURL:    env.MustEnv("GRAFANA_CLOUD_LOKI_URL"),
		LokiUser:   env.MustEnv("GRAFANA_CLOUD_LOKI_USER"),
		LokiAPIKey: env.MustEnv("GRAFANA_CLOUD_LOKI_API_KEY"),
	}
}
