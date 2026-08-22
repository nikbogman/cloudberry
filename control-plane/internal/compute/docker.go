package compute

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const routeLabel = "homelab.route"

// DockerRuntime talks to the local Docker daemon via the official Docker
// Go SDK. client.FromEnv defaults to the local unix socket, same as the
// docker CLI.
type DockerRuntime struct {
	cli *client.Client
}

func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRuntime{cli: cli}, nil
}

// RoutableContainers implements ContainerRuntime. A labeled container is
// only routable if it publishes a port to the host -- routing over the
// Docker bridge network directly would need a second label to disambiguate
// which exposed port to use, which nothing has needed yet.
func (d *DockerRuntime) RoutableContainers() ([]RoutableContainer, error) {
	containers, err := d.cli.ContainerList(context.Background(), container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", routeLabel)),
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	return routableFromContainers(containers), nil
}

func routableFromContainers(containers []types.Container) []RoutableContainer {
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
