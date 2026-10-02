# Cloudberry

Monorepo for my homelab setup.

## Name

Every device is a berry: `raspberry` (Raspberry Pi Zero, Raspbian Lite) and
`blackberry` (Ryzen 3 3250U, 8GB RAM, Ubuntu Server). `cloudberry` is the
whole thing.

## Topology

Two devices on one tailnet:

- **raspberry** — always-on, Wi-Fi, low power. So far it only sends WoL to
  blackberry.
- **blackberry** — the workhorse, Ethernet. Can be suspended, and wakes on
  raspberry's WoL packet (same L2 segment, which is why the packet lands).

The one way in from outside the tailnet is Edge, on Railway — currently
with no authentication. Behind it, access control is tailnet membership. One
`task` run from a dev machine converges both devices.

| Path | What |
|---|---|
| [cmd/](cmd/), [internal/](internal/), [ui/](ui/) | The Platform: wake and suspend the Sleeper — Edge, Waker API, Sleeper API, UI |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Platform design decisions, data flow, constraints |
| [stacks/](stacks/) | Workloads on blackberry, plain Docker Compose behind Caddy |
| [Taskfile.yml](Taskfile.yml) | Deploy: one `task` run converges every device |
| [CONTEXT.md](CONTEXT.md) | Domain glossary — the vocabulary everything else uses |
| [docs/](docs/) | Per-surface docs (edge, waker, sleeper, ui) and [deploy.md](docs/deploy.md) |
| [docs/agents/](docs/agents/) | Agent-tooling contract (issue tracker, triage labels, domain docs) |

One `.env` (from `.env.example`) holds every secret and address; `task` loads it
— see [docs/deploy.md](docs/deploy.md#setup).

## Layout

```
cmd/edge-api/            Edge entrypoint (Railway)
cmd/waker-api/           Waker API entrypoint
cmd/sleeper-api/         Sleeper API entrypoint
internal/edge/           Edge logic
internal/waker/          Waker API logic
internal/sleeper/        Sleeper API logic
internal/tailnet/        shared: identity-header auth, bind-safety
internal/eventlog/       shared: Grafana Cloud event logging
internal/httpresponse/   shared: JSON response helpers
internal/env/            shared: env-var lookup helpers
ui/                      browser SPA (static HTML/JS/CSS, no build step);
                         ui.go is just its go:embed declaration
stacks/                  Compose workloads on blackberry
docs/                    one file per surface, plus deploy.md
deployments/edge-api/    Edge Dockerfile (Railway)
Taskfile.yml             Deploy
```

## Development

```sh
go vet ./... && go test ./...
```

Run a binary locally with its environment, e.g.:

```sh
SLEEPER_MAC_ADDRESS=AA:BB:CC:DD:EE:FF \
GRAFANA_CLOUD_LOKI_URL=https://logs-prod-000.grafana.net/loki/api/v1/push \
GRAFANA_CLOUD_LOKI_USER=123456 GRAFANA_CLOUD_LOKI_API_KEY=glc_xxx \
go run ./cmd/waker-api
```

## Deploying

```sh
task --dry    # print what would run
task          # converge every device
task waker    # or: sleeper, docker, tailscale
```

See [docs/deploy.md](docs/deploy.md). Edge deploys separately, via Railway —
see [docs/edge.md](docs/edge.md#deployment).

## Adding a service

A Go binary goes in `cmd/<name>/` + `internal/<name>/`, with a task in
`Taskfile.yml` reusing `_service`. A Compose workload goes in `stacks/` — see
[stacks/README.md](stacks/README.md).
