package gateway

import (
	"encoding/json"
	"fmt"
	"net"
)

// RouteFromCompose builds a stack's Route from `docker compose config
// --format json`: the route fields from the top-level x-gateway extension,
// the name from the project, and the upstream from the chosen service's first
// published port. ok is false for a stack with no x-gateway, which isn't routed.
func RouteFromCompose(data []byte) (r Route, ok bool, err error) {
	var c struct {
		Name    string `json:"name"`
		Gateway *struct {
			Route
			// Service whose port is the upstream; optional with one service.
			Service string `json:"service"`
		} `json:"x-gateway"`
		Services map[string]struct {
			Ports []struct {
				HostIP    string `json:"host_ip"`
				Published string `json:"published"`
			} `json:"ports"`
		} `json:"services"`
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return r, false, err
	}
	if c.Gateway == nil {
		return r, false, nil
	}
	svc := c.Gateway.Service
	if svc == "" && len(c.Services) == 1 {
		for name := range c.Services {
			svc = name
		}
	}
	s, found := c.Services[svc]
	if !found {
		return r, false, fmt.Errorf("stack %q: x-gateway.service must name one of its services", c.Name)
	}
	// Gated stacks bind their LAN IP, never all interfaces: stack-firewall
	// only lets raspberry through, and the upstream has to be that address.
	if len(s.Ports) == 0 || s.Ports[0].HostIP == "" || s.Ports[0].HostIP == "0.0.0.0" {
		return r, false, fmt.Errorf("stack %q: service %q must publish a port bound to blackberry's LAN IP", c.Name, svc)
	}
	r = c.Gateway.Route
	r.Name = c.Name
	r.Upstream = "http://" + net.JoinHostPort(s.Ports[0].HostIP, s.Ports[0].Published)
	return r, true, nil
}
