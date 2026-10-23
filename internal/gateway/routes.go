package gateway

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Route is one stack exposed through the Router. Port is the loopback listen
// port, and also the HTTPS port `tailscale serve`/`funnel` publishes it on,
// unless Path is set.
type Route struct {
	Name string `json:"name"`
	Port int    `json:"port"`
	// Path publishes the route under this prefix on 443, beside /wake, instead
	// of on its own port. `tailscale serve` strips the prefix before proxying.
	Path     string `json:"path,omitempty"`
	Upstream string `json:"upstream"`
	// Public routes go on Funnel; the rest are tailnet-only.
	Public bool `json:"public"`
	// Wake routes wake blackberry on request; the rest 502 while it's asleep.
	Wake bool `json:"wake"`
}

// Routes is the routes file, cmd/gateway/routes.json.
type Routes struct {
	IdleMinutes int     `json:"idle_minutes"`
	Routes      []Route `json:"routes"`
}

// Funnel only publishes on these ports.
var funnelPorts = map[int]bool{8443: true, 10000: true}

func LoadRoutes(path string) (Routes, error) {
	var cfg Routes
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.IdleMinutes <= 0 {
		return cfg, fmt.Errorf("%s: idle_minutes must be positive", path)
	}
	seen := map[int]bool{}
	seenPath := map[string]bool{}
	for _, r := range cfg.Routes {
		if r.Port == 443 {
			return cfg, fmt.Errorf("%s: route %q: port 443 is the gateway's own tailscale serve, for /wake", path, r.Name)
		}
		if seen[r.Port] {
			return cfg, fmt.Errorf("%s: port %d used twice", path, r.Port)
		}
		seen[r.Port] = true
		if u, err := url.Parse(r.Upstream); err != nil || u.Scheme == "" || u.Host == "" {
			return cfg, fmt.Errorf("%s: route %q: upstream must be an absolute URL", path, r.Name)
		}
		if r.Path != "" {
			if !strings.HasPrefix(r.Path, "/") || r.Path == "/" || strings.HasSuffix(r.Path, "/") {
				return cfg, fmt.Errorf("%s: route %q: path must look like /name; / is /wake's", path, r.Name)
			}
			// Funnel is per port: turning it on for 443 would make /wake public too.
			if r.Public {
				return cfg, fmt.Errorf("%s: route %q: a path route can't be public", path, r.Name)
			}
			if seenPath[r.Path] {
				return cfg, fmt.Errorf("%s: path %s used twice", path, r.Path)
			}
			seenPath[r.Path] = true
		}
		if r.Public && !funnelPorts[r.Port] {
			return cfg, fmt.Errorf("%s: route %q: Funnel only serves ports 8443, 10000 here", path, r.Name)
		}
	}
	return cfg, nil
}
