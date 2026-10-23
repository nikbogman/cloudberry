package gateway

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

// Route is one stack exposed through the Router. Port is both the loopback
// listen port and the HTTPS port `tailscale serve`/`funnel` publishes it on.
type Route struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
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
		if r.Public && !funnelPorts[r.Port] {
			return cfg, fmt.Errorf("%s: route %q: Funnel only serves ports 8443, 10000 here", path, r.Name)
		}
	}
	return cfg, nil
}
