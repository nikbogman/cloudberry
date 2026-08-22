package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const routeLabel = "homelab.route"

// DefaultCPUBaselinePercent is the idle watcher's container-activity
// threshold -- below this counts as idle noise, not real use. An operating
// parameter per the spec, so callers can override it (main.go reads
// COMPUTE_API_CPU_BASELINE_PERCENT) rather than needing a code change.
const DefaultCPUBaselinePercent = 2.0

// DockerRuntime talks to the local Docker daemon via the official Docker
// Go SDK. client.FromEnv defaults to the local unix socket, same as the
// docker CLI.
type DockerRuntime struct {
	cli                *client.Client
	cpuBaselinePercent float64
}

func NewDockerRuntime(cpuBaselinePercent float64) (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRuntime{cli: cli, cpuBaselinePercent: cpuBaselinePercent}, nil
}

// RoutableContainers implements ContainerRuntime. A labeled container is
// only routable if it publishes a port to the host -- routing over the
// Docker bridge network directly would need a second label to disambiguate
// which exposed port to use, which nothing has needed yet.
func (d *DockerRuntime) RoutableContainers() ([]RoutableContainer, error) {
	containers, err := d.labeledContainers()
	if err != nil {
		return nil, err
	}
	return routableFromContainers(containers), nil
}

func (d *DockerRuntime) labeledContainers() ([]container.Summary, error) {
	res, err := d.cli.ContainerList(context.Background(), client.ContainerListOptions{
		Filters: make(client.Filters).Add("label", routeLabel),
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	return res.Items, nil
}

// ActivityAboveBaseline implements ContainerRuntime.
//
// ponytail: CPU only -- network I/O is a cumulative counter, not a rate, so
// a network-based signal needs a per-container previous-sample tracked
// across polls, and nothing routable today needs it. Add that tracking
// here if a low-CPU/high-network workload (e.g. a large file transfer with
// no CPU load) ever needs to hold Compute awake through this signal rather
// than a Hold.
func (d *DockerRuntime) ActivityAboveBaseline() (bool, error) {
	containers, err := d.labeledContainers()
	if err != nil {
		return false, err
	}
	for _, c := range containers {
		percent, err := d.cpuPercent(c.ID)
		if err != nil {
			return false, fmt.Errorf("reading stats for container %s: %w", c.ID, err)
		}
		if percent > d.cpuBaselinePercent {
			return true, nil
		}
	}
	return false, nil
}

// cpuPercent fetches one stats sample and reduces it to a CPU-percent
// figure, from a single one-shot ContainerStats call with
// IncludePreviousSample: the daemon itself collects a CPU sample a second
// apart and populates cpu_stats/precpu_stats accordingly, so no delta state
// needs to be tracked here across polls the way network delta would need.
func (d *DockerRuntime) cpuPercent(containerID string) (float64, error) {
	result, err := d.cli.ContainerStats(context.Background(), containerID, client.ContainerStatsOptions{IncludePreviousSample: true})
	if err != nil {
		return 0, err
	}
	defer result.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(result.Body).Decode(&stats); err != nil {
		return 0, fmt.Errorf("decoding stats: %w", err)
	}

	return cpuPercentFromStats(stats), nil
}

// cpuPercentFromStats is the same CPU-percent formula `docker stats`
// shows, split out from cpuPercent as a pure function so the branch logic
// (no-prior-sample yet, missing online-CPU count) is unit-testable without
// a Docker daemon.
func cpuPercentFromStats(stats container.StatsResponse) float64 {
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	if systemDelta <= 0 {
		return 0
	}
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)

	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}

	return (cpuDelta / systemDelta) * onlineCPUs * 100
}

func routableFromContainers(containers []container.Summary) []RoutableContainer {
	var routable []RoutableContainer
	for _, c := range containers {
		path, ok := c.Labels[routeLabel]
		if !ok {
			continue
		}
		if len(c.Ports) == 0 || c.Ports[0].PublicPort == 0 {
			log.Printf("compute: container %s carries homelab.route=%s but publishes no host port, skipping", c.ID, path)
			continue
		}
		routable = append(routable, RoutableContainer{
			Path: path,
			Addr: net.JoinHostPort("127.0.0.1", strconv.Itoa(int(c.Ports[0].PublicPort))),
		})
	}
	return routable
}
