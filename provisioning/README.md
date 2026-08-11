# pyinfra provisioning

Declarative provisioning for the homelab control plane. Converges the Gateway (`gateway`, the Pi Zero) and the compute host (`compute`) to their declared state: Tailscale joined, Docker installed on `compute`, and both the Gateway API and Compute API (plus the UI's static build) deployed as systemd services. Domain vocabulary (Deploy, Deploy file, Host group) is in [`CONTEXT.md`](../CONTEXT.md).

## Layout

```
provisioning/
  inventory.py                    # gateway / compute / test Host groups
  common.py                       # shared linux_codename()/linux_distro_id()/has_device_role()/TailscaleServeStatus helpers
  settings.py                     # pydantic-settings classes -- typed env var config
  deploy_tailscale.py
  deploy_docker.py
  go_deploy.py                    # shared off-device Go build+ship+systemd helper
  deploy_gateway.py               # cross-compiles + ships the Gateway API binary and the UI's static build
  deploy_compute_api.py
  deploy.py                       # entrypoint composing everything
  deploy.sh                       # wrapper: loads ../.env, resolves short Deploy-file names
  templates/
    binary.service.j2              # systemd unit template, the Gateway API and Compute API

../deploy.sh                      # repo-root forwarder -- run from anywhere, see below
../.env.example                    # checked-in template -- copy to ../.env (gitignored)
```

Run `./deploy.sh` from the repo root or from inside `provisioning/` — both resolve to the same script. Deploy files import from sibling modules and resolve `templates/*.j2` relative to cwd, so `deploy.sh` always `cd`s into `provisioning/` first.

## Setup

An independent [uv](https://docs.astral.sh/uv/) project, not an installable package -- pyinfra's CLI executes these scripts directly. Also requires a Go toolchain on the dev machine:

```sh
cd provisioning
uv sync
```

## Configuration

Real addresses, secrets, and deployment refs are read from the dev machine's environment at Deploy time via typed [`pydantic-settings`](https://docs.pydantic.dev/latest/concepts/pydantic_settings/) classes in `settings.py`. A class with a required field (no default) raises a `ValidationError` listing every missing var at once. Deploy files construct settings lazily inside `has_device_role(...)`, so running one in isolation never demands env vars an unrelated one needs.

Copy [`../.env.example`](../.env.example) to `../.env` (repo root, gitignored) and fill in every required variable below. Run deploys via `./deploy.sh`, not `uv run pyinfra` directly -- it loads `.env` into its own subprocess, so secrets never touch your interactive shell.

### inventory.py (`InventorySettings`)

| Variable | Default | Purpose |
|---|---|---|
| `GATEWAY_HOST`, `GATEWAY_SSH_USER` | `pi-zero.tailnet`, `pi` | Gateway's SSH target |
| `COMPUTE_HOST`, `COMPUTE_SSH_USER` | `main-server.tailnet`, `admin` | Compute's SSH target |
| `GATEWAY_TAILNET_HOST` | `pi-zero.your-tailnet-name.ts.net` | Gateway's Tailscale MagicDNS name; derives `UI_ORIGIN` |
| `COMPUTE_TAILNET_HOST` | `main-server.your-tailnet-name.ts.net` | Compute's Tailscale MagicDNS name; derives `VITE_COMPUTE_API_URL` |
| `GATEWAY_TEST_HOST`, `GATEWAY_TEST_SSH_PORT`, `GATEWAY_TEST_SSH_USER` | `localhost`, `2201`, `root` | `test` group's gateway stand-in (see Testing) |
| `COMPUTE_TEST_HOST`, `COMPUTE_TEST_SSH_PORT`, `COMPUTE_TEST_SSH_USER` | `localhost`, `2202`, `root` | `test` group's compute stand-in |

`GATEWAY_HOST`/`COMPUTE_HOST` are pure SSH targets, used even by `deploy_tailscale.py` itself — on a fresh device, use a plain LAN address, not a tailnet name.

`GATEWAY_TAILNET_HOST`/`COMPUTE_TAILNET_HOST` are separate from the SSH targets: `deploy_compute_api.py`/`deploy_gateway.py` derive the *other* device's browser-facing origin from these, since only the real MagicDNS name gets a valid `tailscale serve` HTTPS cert.

### deploy_tailscale.py (`TailscaleSettings`)

Bold variables are required.

| Variable | Default | Purpose |
|---|---|---|
| **`TAILSCALE_AUTH_KEY`** | — | Tailnet auth key, never committed; only required on first join |

### deploy_gateway.py (`GatewaySettings`, `GatewaySecrets`)

Cross-compiles `control-plane/cmd/gateway` for the Pi Zero W (`GOARCH=arm GOARM=6`) and ships only the binary. The Gateway API does static serving, WoL, and proxying in one process (see [`ARCHITECTURE.md`](../ARCHITECTURE.md)), so this is the only Deploy file for the device.

Bold variables are required.

| Variable | Default | Purpose |
|---|---|---|
| `GATEWAY_HOST`/`_PORT` | `127.0.0.1`, `5000` | Bind address |
| **`COMPUTE_MAC_ADDRESS`** | — | WoL target MAC |
| `COMPUTE_HOST` | `main-server.tailnet` | Host `/server*` forwards to |
| **`COMPUTE_PROXY_PORT`** | — | Port on `COMPUTE_HOST`; no default since the downstream proxy doesn't exist yet |
| **`GRAFANA_CLOUD_LOKI_URL`** | — | Grafana Cloud's Loki push endpoint |
| **`GRAFANA_CLOUD_LOKI_USER`** | — | Loki basic-auth username (numeric instance ID) |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | — | Grafana Cloud Access Policy token, scoped to `logs:write` |

`VITE_COMPUTE_API_URL` is derived from `InventorySettings`, not a separate secret.

After the systemd unit, this file also runs `tailscale serve --bg --https=443 localhost:$GATEWAY_PORT`, guarded by `common.TailscaleServeStatus` so a second run is a no-op. `tailscale serve`'s CLI/JSON shape has moved across Tailscale versions — re-verify against the installed version before trusting it on a new device.

### deploy_compute_api.py (`ComputeApiSettings`, `ComputeApiSecrets`)

Cross-compiles `control-plane/cmd/compute-api` and ships only the binary, same as the Gateway. `GOARCH` is read from the device's real architecture (`common.DpkgArchitecture`), since `compute` isn't a fixed known device.

Bold variables are required.

| Variable | Default | Purpose |
|---|---|---|
| `COMPUTE_API_HOST`/`_PORT` | `127.0.0.1`, `5000` | Bind address |
| **`GRAFANA_CLOUD_LOKI_URL`** | — | Grafana Cloud's Loki push endpoint |
| **`GRAFANA_CLOUD_LOKI_USER`** | — | Loki basic-auth username |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | — | Grafana Cloud Access Policy token, scoped to `logs:write` |

`UI_ORIGIN` (the CORS allow-list entry) is derived from `InventorySettings`, not a separate secret. Also runs `tailscale serve` for `$COMPUTE_API_PORT`, same mechanism and version caveat as the Gateway above.

## Running a Deploy

`./deploy.sh` loads `../.env` and execs pyinfra with `inventory.py` plus whatever args you pass. No target defaults to `deploy.py` (everything); a short name resolves to its `deploy_<name>.py` file (e.g. `gateway` → `deploy_gateway.py`):

```sh
# Everything, both groups
./deploy.sh

# Preview only -- connects and diffs, mutates nothing
./deploy.sh --dry

# One Deploy file against one Host group
./deploy.sh gateway --limit gateway
./deploy.sh gateway --limit gateway --dry
```

pyinfra 3.x has no `--check` flag — use `--dry`. `deploy.py` is never invoked automatically — no CI, cron, or git hook runs it.

## Testing

No pytest/mypy seam applies to infrastructure code, so these checks run through pyinfra's own execution engine.

### Tier 1 — `--dry`

Connects for real and gathers facts, but executes nothing. Catches template/logic errors early; per pyinfra's own caveat it's a prepare-phase estimate, not a correctness proof.

### Tier 2 — disposable containers (the `test` Host group)

`inventory.py`'s `test` group points at long-lived, **systemd-capable** containers over plain `@ssh` — not pyinfra's `@docker` connector, since `@docker` image-mode containers run no init system and can't run `systemd.service` operations. `@ssh` exercises the same connector path `gateway`/`compute` use.

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
```

```sh
./deploy.sh --limit test
```

`device_role` host data makes `gateway-test`/`compute-test` behave like `gateway`/`compute` for gating, so every Deploy file runs against both. `deploy_tailscale.py` skips the actual `tailscale up` join for `test`, so a container never enrolls in the real tailnet.

### Tier 3 — idempotency, the actual correctness check

Run the same Deploy twice in a row against **both** `test` and the real `gateway`/`compute`:

```sh
./deploy.sh --limit test
./deploy.sh --limit test   # again, immediately

./deploy.sh --limit gateway
./deploy.sh --limit gateway     # again, immediately
```

**Pass condition**: the second run's `Grand total` row has an empty `Success` column, everything in `No Change`. Add `--json` to the second run for a scripted gate instead of eyeballing the table.

## Known gaps

- **Tiers 2/3 haven't run against real infrastructure** — the `tailscale serve` automation for both Deploy files is unverified against a real device; re-check the CLI/JSON version caveat above before trusting it blindly.
- **Bind-address enforcement** ("never bound off-tailnet") isn't checked by pyinfra.
- **The server-side workload proxy** `/server*` forwards to doesn't exist yet — a manual prerequisite, out of pyinfra's scope.
