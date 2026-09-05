# Deploy

Declarative provisioning with [pyinfra](https://pyinfra.com/). One run converges
the Gateway and the compute host to their declared state: Tailscale joined,
Docker installed on `compute`, and both binaries (plus the UI's static build)
deployed as systemd services.

Per-service deploy details live with the service: [`gateway.md`](gateway.md),
[`compute.md`](compute.md). Vocabulary (Deploy, Deploy file, Host group) is in
[`CONTEXT.md`](../CONTEXT.md).

## Layout

```
deploy/
  inventory.py          gateway / compute / test Host groups
  common.py             shared helpers: linux_codename/linux_distro_id/
                        has_device_role, DpkgArchitecture, TailscaleServeStatus
  settings.py           pydantic-settings classes -- typed env var config
  go_build.py           shared off-device Go build+ship+systemd helper
  deploy_tailscale.py   ┐
  deploy_docker.py      │ one Deploy file per piece of infrastructure or app
  deploy_gateway.py     │
  deploy_compute_api.py ┘
  deploy.py             entrypoint composing everything
  deploy.sh             wrapper: loads ../.env, resolves short Deploy-file names
  templates/
    binary.service.j2   systemd unit template, both binaries
```

`../deploy.sh` is a repo-root forwarder to the same script. Deploy files import
from sibling modules and resolve `templates/*.j2` relative to cwd, so
`deploy.sh` always `cd`s into `deploy/` first.

## Setup

An independent [uv](https://docs.astral.sh/uv/) project, not an installable
package — pyinfra's CLI executes these scripts directly. Also requires a Go
toolchain and Node on the dev machine.

```sh
cd deploy
uv sync
```

Copy `.env.example` to `.env` at the repo root (gitignored) and fill in every
required variable. Run deploys via `./deploy.sh`, not `uv run pyinfra` directly
— it loads `.env` into its own subprocess, so secrets never touch your
interactive shell.

## Configuration

Real addresses, secrets, and deployment refs are read from the dev machine's
environment at Deploy time via typed
[`pydantic-settings`](https://docs.pydantic.dev/latest/concepts/pydantic_settings/)
classes in `settings.py`. A class with a required field (no default) raises a
`ValidationError` listing every missing var at once. Deploy files construct
settings lazily inside `has_device_role(...)`, so running one in isolation never
demands env vars an unrelated one needs.

### Inventory (`InventorySettings`)

| Variable | Default | Purpose |
|---|---|---|
| `GATEWAY_HOST`, `GATEWAY_SSH_USER` | `pi-zero.tailnet`, `pi` | Gateway's SSH target |
| `COMPUTE_HOST`, `COMPUTE_SSH_USER` | `main-server.tailnet`, `admin` | Compute's SSH target |
| `GATEWAY_TAILNET_HOST` | `pi-zero.your-tailnet-name.ts.net` | Gateway's Tailscale MagicDNS name; derives `UI_ORIGIN` |
| `COMPUTE_TAILNET_HOST` | `main-server.your-tailnet-name.ts.net` | Compute's Tailscale MagicDNS name; derives `VITE_COMPUTE_API_URL` |
| `GATEWAY_TEST_HOST`, `GATEWAY_TEST_SSH_PORT`, `GATEWAY_TEST_SSH_USER` | `localhost`, `2201`, `root` | `test` group's gateway stand-in |
| `COMPUTE_TEST_HOST`, `COMPUTE_TEST_SSH_PORT`, `COMPUTE_TEST_SSH_USER` | `localhost`, `2202`, `root` | `test` group's compute stand-in |

`GATEWAY_HOST`/`COMPUTE_HOST` are pure SSH targets, used even by
`deploy_tailscale.py` itself — on a fresh device, use a plain LAN address, not a
tailnet name.

`GATEWAY_TAILNET_HOST`/`COMPUTE_TAILNET_HOST` are separate from the SSH targets:
each service's Deploy file derives the *other* device's browser-facing origin
from these, since only the real MagicDNS name gets a valid `tailscale serve`
HTTPS cert.

### Tailscale (`TailscaleSettings`)

| Variable | Default | Purpose |
|---|---|---|
| **`TAILSCALE_AUTH_KEY`** | — | Tailnet auth key, never committed; only required on first join |

Service-specific variables are in [`gateway.md`](gateway.md#deploy-time-variables)
and [`compute.md`](compute.md#deploy-time-variables).

## Running a Deploy

`./deploy.sh` loads `.env` and execs pyinfra with `inventory.py` plus whatever
args you pass. No target defaults to `deploy.py` (everything); a short name
resolves to its `deploy_<name>.py` file (e.g. `gateway` → `deploy_gateway.py`):

```sh
./deploy.sh                              # everything, both groups
./deploy.sh --dry                        # preview: connects and diffs, mutates nothing
./deploy.sh gateway --limit gateway      # one Deploy file against one Host group
./deploy.sh gateway --limit gateway --dry
```

pyinfra 3.x has no `--check` flag — use `--dry`. `deploy.py` is never invoked
automatically: no CI, cron, or git hook runs it.

## Testing

No pytest/mypy seam applies to infrastructure code, so these checks run through
pyinfra's own execution engine.

### Tier 1 — `--dry`

Connects for real and gathers facts, but executes nothing. Catches
template/logic errors early; per pyinfra's own caveat it's a prepare-phase
estimate, not a correctness proof.

### Tier 2 — disposable containers (the `test` Host group)

`inventory.py`'s `test` group points at long-lived, **systemd-capable**
containers over plain `@ssh` — not pyinfra's `@docker` connector, since
`@docker` image-mode containers run no init system and can't run
`systemd.service` operations. `@ssh` exercises the same connector path
`gateway`/`compute` use.

```sh
docker run -d --name gateway-test --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 2201:22 \
  jrei/systemd-debian:12
docker run -d --name compute-test --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 2202:22 \
  jrei/systemd-debian:12

# Install your pyinfra dev machine's public key + an SSH server on each
# (jrei/systemd-debian ships openssh-server disabled -- enable and seed
# authorized_keys via `docker exec`, then `systemctl start ssh`).

./deploy.sh --limit test
```

`device_role` host data makes `gateway-test`/`compute-test` behave like
`gateway`/`compute` for gating, so every Deploy file runs against both.
`deploy_tailscale.py` skips the actual `tailscale up` join for `test`, so a
container never enrolls in the real tailnet.

### Tier 3 — idempotency, the actual correctness check

Run the same Deploy twice in a row against **both** `test` and the real
`gateway`/`compute`:

```sh
./deploy.sh --limit test
./deploy.sh --limit test        # again, immediately

./deploy.sh --limit gateway
./deploy.sh --limit gateway     # again, immediately
```

**Pass condition**: the second run's `Grand total` row has an empty `Success`
column, everything in `No Change`. Add `--json` to the second run for a scripted
gate instead of eyeballing the table.

## Known gaps

- **Tiers 2/3 haven't run against real infrastructure** — the `tailscale serve`
  automation for both Deploy files is unverified against a real device;
  re-check the CLI/JSON version caveat before trusting it blindly.
- Standing up a labeled workload container is a manual step, outside pyinfra's
  scope.
- **One-time migration**: the Gateway's unit was renamed `gateway.service` →
  `gateway-api.service`. pyinfra installs the new one but does not remove the
  old, so on a device deployed before that rename, both would bind the same
  port. Run `systemctl disable --now gateway && rm /etc/systemd/system/gateway.service`
  on the Pi once, then delete this bullet.

Bind-address enforcement and the system's other limitations are in
[`ARCHITECTURE.md`](../ARCHITECTURE.md#constraints).
