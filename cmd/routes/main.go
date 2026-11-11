// routes prints gateway's routes file, built from the x-route extension in
// each <stacks dir>/*/compose.yml, or with -ports the upstream ports of those
// routes, for stack-firewall. `docker compose config` needs each stack's .env,
// but not Docker or blackberry. hostd does the same on blackberry (GET /routes).
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/nikbogman/metal/internal/routes"
)

func main() {
	ports := flag.Bool("ports", false, "print the routes' upstream ports, space-separated")
	idle := flag.Int("idle-minutes", 30, "default idle_minutes")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("usage: routes [-ports] [-idle-minutes N] <stacks dir>")
	}
	cfg, data, err := routes.FromDir(flag.Arg(0), *idle)
	if err != nil {
		log.Fatal(err)
	}
	if *ports {
		fmt.Println(strings.Join(routes.Ports(cfg), " "))
		return
	}
	fmt.Println(string(data))
}
