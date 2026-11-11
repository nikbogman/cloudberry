package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"path/filepath"
)

// FromCompose builds a stack's Route from `docker compose config
// --format json`: the route fields from the top-level x-route extension,
// the name from the project, and the upstream from the chosen service's first
// published port. ok is false for a stack with no x-route, which isn't routed.
func FromCompose(data []byte) (r Route, ok bool, err error) {
	var c struct {
		Name   string `json:"name"`
		XRoute *struct {
			Route
			// Service whose port is the upstream; optional with one service.
			Service string `json:"service"`
		} `json:"x-route"`
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
	if c.XRoute == nil {
		return r, false, nil
	}
	svc := c.XRoute.Service
	if svc == "" && len(c.Services) == 1 {
		for name := range c.Services {
			svc = name
		}
	}
	s, found := c.Services[svc]
	if !found {
		return r, false, fmt.Errorf("stack %q: x-route.service must name one of its services", c.Name)
	}
	// Gated stacks bind their LAN IP, never all interfaces: stack-firewall
	// only lets raspberry through, and the upstream has to be that address.
	if len(s.Ports) == 0 || s.Ports[0].HostIP == "" || s.Ports[0].HostIP == "0.0.0.0" {
		return r, false, fmt.Errorf("stack %q: service %q must publish a port bound to blackberry's LAN IP", c.Name, svc)
	}
	r = c.XRoute.Route
	r.Name = c.Name
	r.Upstream = "http://" + net.JoinHostPort(s.Ports[0].HostIP, s.Ports[0].Published)
	return r, true, nil
}

// FromDir builds the routes file from every <dir>/*/compose.yml, with
// idleMinutes as the default idle time. `docker compose config` needs each
// stack's .env, since it resolves the interpolation, but not a Docker daemon.
func FromDir(dir string, idleMinutes int) (Routes, []byte, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*", "compose.yml"))
	if err != nil || len(files) == 0 {
		return Routes{}, nil, fmt.Errorf("no */compose.yml in %s", dir)
	}
	cfg := Routes{IdleMinutes: idleMinutes, Routes: []Route{}}
	for _, f := range files {
		var stderr bytes.Buffer
		cmd := exec.Command("docker", "compose", "-f", f, "config", "--format", "json")
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return Routes{}, nil, fmt.Errorf("%s: %v: %s", f, err, bytes.TrimSpace(stderr.Bytes()))
		}
		r, ok, err := FromCompose(out)
		if err != nil {
			return Routes{}, nil, fmt.Errorf("%s: %w", f, err)
		}
		if ok {
			cfg.Routes = append(cfg.Routes, r)
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return Routes{}, nil, err
	}
	parsed, err := Parse(data, "generated routes")
	return parsed, data, err
}

// Ports is the upstream port of each route, for stack-firewall's PORTS.
func Ports(cfg Routes) []string {
	ps := []string{}
	for _, r := range cfg.Routes {
		u, _ := url.Parse(r.Upstream) // Parse checked it
		ps = append(ps, u.Port())
	}
	return ps
}
