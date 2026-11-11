package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/nikbogman/cloudberry/internal/cors"
	"github.com/nikbogman/cloudberry/internal/eventlog"
	"github.com/nikbogman/cloudberry/internal/gateway"
	"github.com/nikbogman/cloudberry/internal/routes"
	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// version is set at build time (-ldflags "-X main.version=..."): a release
// tag for release builds, which update themselves; anything else doesn't.
var version = "dev"

func main() {
	log.Printf("gateway %s", version)
	cfg := gateway.ConfigFromEnv()
	rs, err := routes.Load(cfg.RoutesFile)
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
	router := gateway.NewRouter(cfg.HostdURL, cfg.MACAddress, sender, logger)
	if err := router.LoadState(cfg.StateFile); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/settings", tailnet.RequireTailnetIdentity(router.SettingsHandler()))
	addr := net.JoinHostPort(cfg.BindHost, cfg.Port)
	log.Printf("gateway listening on %s", addr)
	go func() { log.Fatal(http.ListenAndServe(addr, cors.Allow(cfg.UIOrigin, mux))) }()

	for _, r := range rs.Routes {
		h, err := router.RouteHandler(r)
		if err != nil {
			log.Fatal(err)
		}
		if r.Path == "" {
			log.Printf("gateway route %s on tsnet host %s -> %s", r.Name, r.Name, r.Upstream)
			go func() { log.Fatal(gateway.ServeHostname(r, cfg.TsnetDir, h)) }()
			continue
		}
		// Loopback only: tailscaled is the sole way in.
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(r.Port))
		log.Printf("gateway route %s on %s -> %s", r.Name, addr, r.Upstream)
		go func() { log.Fatal(http.ListenAndServe(addr, h)) }()
	}
	if err := gateway.Expose(rs.Routes); err != nil {
		log.Fatal(err)
	}

	go router.RunIdleSuspend(context.Background())

	// Returns when the routes or the binary changed: exit, and systemd
	// (Restart=always) starts gateway again on the new ones.
	(&gateway.Updates{
		HostdURL:   cfg.HostdURL,
		RoutesFile: cfg.RoutesFile,
		Current:    rs,
		Binary:     cfg.Update.Updater("gateway", version),
		Router:     router,
		Logger:     logger,
		Every:      cfg.Update.Every,
	}).Run(context.Background())
	log.Print("restarting for an update")
}
