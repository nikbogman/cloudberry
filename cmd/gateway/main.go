package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
	for _, r := range rs.Routes {
		if r.Name == cfg.Hostname {
			log.Fatalf("route %q has gateway's own hostname", r.Name)
		}
	}

	grafanaCfg := eventlog.MustGrafanaConfig()
	logger := eventlog.NewGrafanaCloudLogger(grafanaCfg.LokiURL, grafanaCfg.LokiUser, grafanaCfg.LokiAPIKey, "gateway")
	sender := gateway.NewWakeOnLanSender()
	router := gateway.NewRouter(cfg.HostdURL, cfg.MACAddress, sender, logger)
	if err := router.LoadState(cfg.StateFile); err != nil {
		log.Fatal(err)
	}

	cfg.Config.OnWake = router.WakeRequested
	handler, err := gateway.NewHandler(cfg.Config, sender, logger)
	if err != nil {
		log.Fatal(err)
	}

	// gateway's own tailnet node (raspberry.<tailnet>.ts.net): the Pi runs no
	// tailscaled, so this node is both how the UI reaches /wake and /settings
	// and how gateway reaches hostd.
	node := gateway.Node(cfg.Hostname, cfg.TsnetDir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	tsStatus, err := node.Up(ctx)
	if err != nil {
		log.Fatalf("tsnet node %s: %v", cfg.Hostname, err)
	}
	cancel()
	tailnetClient := node.HTTPClient()
	router.SetHTTPClient(tailnetClient)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/settings", tailnet.RequireTailnetIdentity(router.SettingsHandler()))
	mux.Handle("GET /status", tailnet.RequireTailnetIdentity(router.StatusHandler()))

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
		// The stack sees the path without its prefix; /<path> redirects to /<path>/.
		log.Printf("gateway route %s on %s%s -> %s", r.Name, cfg.Hostname, r.Path, r.Upstream)
		mux.Handle(r.Path+"/", http.StripPrefix(r.Path, h))
	}
	// "raspberry.tail1234.ts.net." -> "tail1234.ts.net", for the routes' URLs.
	if _, domain, ok := strings.Cut(strings.TrimSuffix(tsStatus.Self.DNSName, "."), "."); ok {
		router.SetURLs(cfg.Hostname, domain)
	}
	// Every upstream is on blackberry, so the first one tells whether it's up.
	if len(rs.Routes) > 0 {
		u, err := url.Parse(rs.Routes[0].Upstream)
		if err != nil {
			log.Fatal(err)
		}
		go router.RunProbe(context.Background(), u.Host, 15*time.Second)
	}

	log.Printf("gateway serving https://%s on the tailnet", cfg.Hostname)
	go func() { log.Fatal(gateway.ServeNode(node, false, cors.Allow(cfg.UIOrigin, mux))) }()
	// Loopback too, for /wake from a shell on raspberry (no identity there).
	addr := net.JoinHostPort(cfg.BindHost, cfg.Port)
	log.Printf("gateway listening on %s", addr)
	go func() { log.Fatal(http.ListenAndServe(addr, cors.Allow(cfg.UIOrigin, mux))) }()

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
		Client:     &http.Client{Transport: tailnetClient.Transport, Timeout: 10 * time.Second},
	}).Run(context.Background())
	// Exit 0 straight away: closing the node first would make the listener
	// goroutines log.Fatal (exit 1), which systemd counts as a failure.
	log.Print("restarting for an update")
	os.Exit(0)
}
