# Cloudberry

Monorepo for my homelab setup.

## Name

Every device is a berry: `raspberry` (Raspberry Pi Zero, Raspbian Lite) and
`blackberry` (Ryzen 3 3250U, 8GB RAM, Ubuntu Server). `cloudberry` is the
whole thing.

## Topology

Two devices on one tailnet, no ports exposed to the internet:

- **raspberry** — always-on, Wi-Fi, low power. So far it only sends WoL to
  blackberry.
- **blackberry** — the workhorse, Ethernet. Can be suspended, and wakes on
  raspberry's WoL packet (same L2 segment, which is why the packet lands).

Tailscale is the only way in; access control is tailnet membership. One
pyinfra run from a dev machine converges both.

| Path | What |
|---|---|
| [waker-service/](waker-service/) | Wake and suspend the Sleeper from the tailnet — Waker API, Sleeper API, UI |
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
