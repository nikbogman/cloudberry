package gateway

import (
	"fmt"
	"os/exec"
)

// Expose publishes each route's loopback port on the same HTTPS port:
// Funnel for public routes, `tailscale serve` for the rest.
// ponytail: only adds; a route removed from the file stays published until
// `tailscale serve reset` on raspberry.
func Expose(routes []Route) error {
	for _, r := range routes {
		mode := "serve"
		if r.Public {
			mode = "funnel"
		}
		port := fmt.Sprint(r.Port)
		if out, err := exec.Command("tailscale", mode, "--bg", "--https="+port, "localhost:"+port).CombinedOutput(); err != nil {
			return fmt.Errorf("tailscale %s for route %q: %w: %s", mode, r.Name, err, out)
		}
	}
	return nil
}
