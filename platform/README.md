# Platform

Wakes, monitors, and suspends the Sleeper — the box running the workloads —
so it doesn't run 24/7. Edge (Railway) serves the UI and forwards to the Waker
API (Pi Zero, sends WoL) and the Sleeper API. Components and design:
[ARCHITECTURE.md](ARCHITECTURE.md).

## Docs

| Document | Owns |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Design decisions, data flow, constraints, shared Go packages |
| [docs/edge.md](docs/edge.md) | Edge: routes, tailnet access, environment, deployment |
| [docs/waker.md](docs/waker.md) | Waker API: routes, environment, deployment |
| [docs/sleeper.md](docs/sleeper.md) | Sleeper API: routes, environment, deployment |
| [docs/ui.md](docs/ui.md) | Browser SPA: behavior, development |
| [../CONTEXT.md](../CONTEXT.md) | Domain glossary — the vocabulary everything else uses |
| [../docs/deploy.md](../docs/deploy.md) | pyinfra: inventory, running a Deploy, the three test tiers |

## Layout

```
ARCHITECTURE.md          design reference
docs/                    one file per surface (edge, waker, sleeper, ui)
go.mod                   one Go module for all three binaries
Dockerfile               Edge image, for Railway
cmd/edge-api/            Edge entrypoint
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
.env.example             template -- copy to .env (gitignored)
```

Provisioning lives at the repo root in [../deploy/](../deploy/), shared across
every homelab service, and builds these binaries from here via `module_dir`.

## Development

```sh
# Go, all binaries (from this folder)
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

From the repo root:

```sh
./deploy.sh waker --dry       # preview just the Waker
./deploy.sh sleeper_api       # converge just the Sleeper API
./deploy.sh                   # converge every device
```

This service's Deploy files are grouped in
[`../deploy/platform/`](../deploy/platform/); secrets setup is in
[../docs/deploy.md](../docs/deploy.md#setup). Edge deploys separately, via
Railway — see [docs/edge.md](docs/edge.md#deployment).

## Status

Fully implemented; one command converges both devices, and Edge deploys via
Railway. Outstanding gaps:
[../docs/deploy.md](../docs/deploy.md#known-gaps) and
[ARCHITECTURE.md](ARCHITECTURE.md#constraints).
