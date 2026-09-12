# Homelab

Monorepo for the homelab.

| Path | What |
|---|---|
| [waker-service/](waker-service/) | Wake, monitor, and suspend the Sleeper from the tailnet — Waker API, Sleeper API, UI |
| [deploy/](deploy/) | pyinfra: one run converges every device across every service |
| [CONTEXT.md](CONTEXT.md) | Domain glossary — the vocabulary everything else uses |
| [docs/deploy.md](docs/deploy.md) | Inventory, running a Deploy, the three test tiers |
| [docs/agents/](docs/agents/) | Agent-tooling contract (issue tracker, triage labels, domain docs) |

Each service owns its own code, docs, toolchain, and `.env`; the root `.env`
holds the device addresses and the Tailscale auth key. `deploy/deploy.sh`
sources both — see [docs/deploy.md](docs/deploy.md#setup).

## Deploying

```sh
./deploy.sh --dry    # preview every device
./deploy.sh          # converge every device
```

See [docs/deploy.md](docs/deploy.md).

## Adding a service

A top-level folder with its own docs and `.env`. If it deploys anything, its
Deploy files go in `deploy/<service>/`, get `local.include`d from
`deploy/deploy.py`, and its `.env` gets sourced in `deploy/deploy.sh`. Follow
[waker-service/](waker-service/README.md) as the worked example.
