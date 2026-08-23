package compute

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nikbogman/homelab/control-plane/internal/env"
)

// EnvConfig holds compute-api runtime configuration read from the process
// environment.
type EnvConfig struct {
	Host               string
	Port               string
	UIOrigin           string
	CPUBaselinePercent float64
	IdleTimeout        time.Duration
	AutosuspendDryRun  bool
}

// ConfigFromEnv reads and validates compute-api runtime configuration from
// the process environment. COMPUTE_API_HOST/PORT and the autosuspend
// tuning knobs default to their current production values; UI_ORIGIN names
// the one browser origin CORS should allow, so it has no safe default and
// is required.
func ConfigFromEnv() (EnvConfig, error) {
	cpuBaselinePercent, err := strconv.ParseFloat(env.EnvOr("COMPUTE_API_CPU_BASELINE_PERCENT", strconv.FormatFloat(DefaultCPUBaselinePercent, 'f', -1, 64)), 64)
	if err != nil {
		return EnvConfig{}, fmt.Errorf("COMPUTE_API_CPU_BASELINE_PERCENT: %w", err)
	}

	idleTimeout, err := time.ParseDuration(env.EnvOr("COMPUTE_API_IDLE_TIMEOUT", "1h"))
	if err != nil {
		return EnvConfig{}, fmt.Errorf("COMPUTE_API_IDLE_TIMEOUT: %w", err)
	}

	dryRun, err := strconv.ParseBool(env.EnvOr("COMPUTE_API_AUTOSUSPEND_DRY_RUN", "false"))
	if err != nil {
		return EnvConfig{}, fmt.Errorf("COMPUTE_API_AUTOSUSPEND_DRY_RUN: %w", err)
	}

	return EnvConfig{
		Host:               env.EnvOr("COMPUTE_API_HOST", "127.0.0.1"),
		Port:               env.EnvOr("COMPUTE_API_PORT", "5000"),
		UIOrigin:           env.MustEnv("UI_ORIGIN"),
		CPUBaselinePercent: cpuBaselinePercent,
		IdleTimeout:        idleTimeout,
		AutosuspendDryRun:  dryRun,
	}, nil
}
