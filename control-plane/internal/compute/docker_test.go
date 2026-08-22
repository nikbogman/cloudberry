package compute

import (
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestRoutableFromContainersMatchesByLabel(t *testing.T) {
	containers := []container.Summary{
		{
			Labels: map[string]string{routeLabel: "/immich"},
			Ports:  []container.PortSummary{{PublicPort: 8080}},
		},
	}

	got := routableFromContainers(containers)

	if len(got) != 1 || got[0].Path != "/immich" || got[0].Addr != "127.0.0.1:8080" {
		t.Fatalf("got %+v, want one container routed to /immich on 127.0.0.1:8080", got)
	}
}

func TestRoutableFromContainersSkipsContainersWithoutTheRouteLabel(t *testing.T) {
	containers := []container.Summary{
		{Labels: map[string]string{"some.other.label": "x"}, Ports: []container.PortSummary{{PublicPort: 8080}}},
	}

	if got := routableFromContainers(containers); len(got) != 0 {
		t.Fatalf("got %+v, want no routable containers", got)
	}
}

func TestRoutableFromContainersSkipsALabeledContainerWithNoPublishedPort(t *testing.T) {
	containers := []container.Summary{
		{Labels: map[string]string{routeLabel: "/immich"}, Ports: nil},
	}

	if got := routableFromContainers(containers); len(got) != 0 {
		t.Fatalf("got %+v, want no routable containers (label present but nothing published)", got)
	}
}

func TestCpuPercentFromStatsComputesUsageDeltaAcrossAllOnlineCPUs(t *testing.T) {
	stats := container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 4_000_000_000},
			SystemUsage: 40_000_000_000,
			OnlineCPUs:  2,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 2_000_000_000},
			SystemUsage: 20_000_000_000,
		},
	}

	// cpuDelta/systemDelta * onlineCPUs * 100 = (2e9/20e9) * 2 * 100 = 20%.
	if got := cpuPercentFromStats(stats); got != 20 {
		t.Fatalf("got %v%%, want 20%%", got)
	}
}

func TestCpuPercentFromStatsFallsBackToPercpuUsageLengthWhenOnlineCPUsIsUnset(t *testing.T) {
	stats := container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 4_000_000_000, PercpuUsage: []uint64{0, 0}},
			SystemUsage: 40_000_000_000,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 2_000_000_000},
			SystemUsage: 20_000_000_000,
		},
	}

	if got := cpuPercentFromStats(stats); got != 20 {
		t.Fatalf("got %v%%, want 20%% (falling back to len(PercpuUsage)=2 online CPUs)", got)
	}
}

func TestCpuPercentFromStatsReturnsZeroWithNoPriorSample(t *testing.T) {
	stats := container.StatsResponse{
		// CPUStats and PreCPUStats identical: systemDelta <= 0, as if no
		// prior sample were available to diff against.
		CPUStats:    container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 4_000_000_000}, SystemUsage: 40_000_000_000},
		PreCPUStats: container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 4_000_000_000}, SystemUsage: 40_000_000_000},
	}

	if got := cpuPercentFromStats(stats); got != 0 {
		t.Fatalf("got %v%%, want 0%% with no prior sample to diff against", got)
	}
}
