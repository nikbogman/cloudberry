// routes prints gateway's routes file, built from the x-gateway extension in
// each <stacks dir>/*/compose.yml, or with -ports the upstream ports of those
// routes, for stack-firewall. Runs on the dev machine: `docker compose config`
// needs each stack's .env, but not Docker or blackberry.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nikbogman/cloudberry/internal/gateway"
)

func main() {
	ports := flag.Bool("ports", false, "print the routes' upstream ports, space-separated")
	idle := flag.Int("idle-minutes", 30, "default idle_minutes")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("usage: routes [-ports] [-idle-minutes N] <stacks dir>")
	}
	files, err := filepath.Glob(filepath.Join(flag.Arg(0), "*", "compose.yml"))
	if err != nil || len(files) == 0 {
		log.Fatalf("no */compose.yml in %s", flag.Arg(0))
	}

	cfg := gateway.Routes{IdleMinutes: *idle}
	for _, f := range files {
		cmd := exec.Command("docker", "compose", "-f", f, "config", "--format", "json")
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			log.Fatalf("%s: %v", f, err)
		}
		r, ok, err := gateway.RouteFromCompose(out)
		if err != nil {
			log.Fatalf("%s: %v", f, err)
		}
		if ok {
			cfg.Routes = append(cfg.Routes, r)
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := gateway.ParseRoutes(data, "generated routes"); err != nil {
		log.Fatal(err)
	}

	if *ports {
		var ps []string
		for _, r := range cfg.Routes {
			u, _ := url.Parse(r.Upstream) // ParseRoutes checked it
			ps = append(ps, u.Port())
		}
		fmt.Println(strings.Join(ps, " "))
		return
	}
	fmt.Println(string(data))
}
