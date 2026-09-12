package compute

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// handleProxy routes a request to whichever container's homelab.route
// label prefixes the request path. A real HTTP response from the
// container (even an error status) passes through untouched -- only a
// transport-level failure to reach the container produces a 502, via
// httputil.ReverseProxy's default ErrorHandler. There is no retry/wake
// loop: workload containers run continuously (restart: unless-stopped),
// so a container never needs waking, and if Compute itself is asleep
// this process isn't running to be asked.
func (h *Handler) handleProxy(w http.ResponseWriter, r *http.Request) {
	container, err := matchRoute(r.URL.Path, h.runtime)
	if err != nil {
		http.Error(w, "container discovery failed", http.StatusBadGateway)
		return
	}
	if container == nil {
		http.NotFound(w, r)
		return
	}

	target := &url.URL{Scheme: "http", Host: container.Addr}
	proxy := httputil.NewSingleHostReverseProxy(target)

	http.StripPrefix(container.Path, proxy).ServeHTTP(w, r)
}

// matchRoute finds the routable container whose label prefixes path. A
// nil, nil return means the runtime was consulted successfully but no
// container matches.
func matchRoute(path string, runtime ContainerRuntime) (*RoutableContainer, error) {
	containers, err := runtime.RoutableContainers()
	if err != nil {
		return nil, err
	}

	for i, c := range containers {
		if path == c.Path || strings.HasPrefix(path, c.Path+"/") {
			return &containers[i], nil
		}
	}
	return nil, nil
}
