package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/nikbogman/cloudberry/internal/cors"
	"github.com/nikbogman/cloudberry/internal/eventlog"
	"github.com/nikbogman/cloudberry/internal/gateway"
)

func main() {
	cfg := gateway.ConfigFromEnv()
	routes, err := gateway.LoadRoutes(cfg.RoutesFile)
	if err != nil {
		log.Fatal(err)
	}

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "gateway")
	sender := gateway.NewWakeOnLanSender()

	handler, err := gateway.NewHandler(cfg.Config, sender, logger)
	if err != nil {
		log.Fatal(err)
	}
	addr := net.JoinHostPort(cfg.BindHost, cfg.Port)
	log.Printf("gateway listening on %s", addr)
	go func() { log.Fatal(http.ListenAndServe(addr, cors.Allow(cfg.UIOrigin, handler))) }()

	router := gateway.NewRouter(cfg.HostdURL, cfg.MACAddress, sender, logger)
	for _, r := range routes.Routes {
		h, err := router.RouteHandler(r)
		if err != nil {
			log.Fatal(err)
		}
		// Loopback only: tailscaled is the sole way in.
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(r.Port))
		log.Printf("gateway route %s on %s -> %s", r.Name, addr, r.Upstream)
		go func() { log.Fatal(http.ListenAndServe(addr, h)) }()
	}
	if err := gateway.Expose(routes.Routes); err != nil {
		log.Fatal(err)
	}

	router.RunIdleSuspend(context.Background(), time.Duration(routes.IdleMinutes)*time.Minute)
}
