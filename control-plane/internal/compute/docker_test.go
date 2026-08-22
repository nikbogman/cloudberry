package compute

import (
	"testing"

	"github.com/docker/docker/api/types"
)

func TestRoutableFromContainersMatchesByLabel(t *testing.T) {
	containers := []types.Container{
		{
			Labels: map[string]string{routeLabel: "/immich"},
			Ports:  []types.Port{{PublicPort: 8080}},
		},
	}

	got := routableFromContainers(containers)

	if len(got) != 1 || got[0].Path != "/immich" || got[0].Addr != "127.0.0.1:8080" {
		t.Fatalf("got %+v, want one container routed to /immich on 127.0.0.1:8080", got)
	}
}

func TestRoutableFromContainersSkipsContainersWithoutTheRouteLabel(t *testing.T) {
	containers := []types.Container{
		{Labels: map[string]string{"some.other.label": "x"}, Ports: []types.Port{{PublicPort: 8080}}},
	}

	if got := routableFromContainers(containers); len(got) != 0 {
		t.Fatalf("got %+v, want no routable containers", got)
	}
}

func TestRoutableFromContainersSkipsALabeledContainerWithNoPublishedPort(t *testing.T) {
	containers := []types.Container{
		{Labels: map[string]string{routeLabel: "/immich"}, Ports: nil},
	}

	if got := routableFromContainers(containers); len(got) != 0 {
		t.Fatalf("got %+v, want no routable containers (label present but nothing published)", got)
	}
}
