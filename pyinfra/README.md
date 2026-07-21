# pyinfra provisioning

Declarative provisioning for the homelab control plane, per
`.scratch/pyinfra-provisioning/spec.md`. Converges the Pi Zero (`pi`) and
the main server (`server`) to their declared state: Tailscale joined,
Docker installed on `server`, the full Caddy config templated on `pi`, and
both Control APIs (plus the Control UI's static build) deployed as
systemd services. Domain vocabulary (Deploy, Deploy file, Host group) is
defined in the root [`CONTEXT.md`](../CONTEXT.md); pyinfra-specific
implementation notes and citations live in
[`docs/agents/pyinfra.md`](../docs/agents/pyinfra.md).

## Layout

```
pyinfra/
  inventory.py                    # pi / server / test Host groups (ticket 01)
  common.py                       # shared debian_codename()/has_device_kind() helpers
  deploy_tailscale.py             # ticket 02
  deploy_docker.py                # ticket 03
  control_api_deploy.py           # shared git+systemd @deploy helper (ticket 04)
  deploy_control_pi_api.py        # ticket 05 (+ Control UI static delivery)
  deploy_control_server_api.py    # ticket 06
  deploy_caddy.py                 # ticket 07
  deploy.py                       # entrypoint composing everything (ticket 08)
  templates/
    control-api.service.j2        # systemd unit template, both Control APIs
    Caddyfile.j2                  # the Pi's complete Caddy config
```

**Run every command from inside this directory** (`cd pyinfra/` first).
Deploy files import from sibling modules (`from common import ...`,
`from control_api_deploy import ...`) and `files.template` resolves
`templates/*.j2` relative to the current working directory — both need
`pyinfra/` on `sys.path`/as cwd, which pyinfra only does automatically for
the directory it's actually invoked from. (Running from the repo root
with `pyinfra/`-prefixed paths, as `docs/agents/pyinfra.md`'s generic
examples show, would also put a directory literally named `pyinfra` next
to the real installed `pyinfra` package on `sys.path` — avoided entirely
by the `cd pyinfra/` convention.)

## Setup

An independent [uv](https://docs.astral.sh/uv/) project, like the two
Flask services, but not an installable package (`[tool.uv] package =
false` in `pyproject.toml`) -- it's a directory of scripts pyinfra's CLI
executes, not a Python package anything imports:

```sh
cd pyinfra
uv sync
```

## Configuration

Nothing operator-specific is hardcoded (ticket 01) — real addresses,
secrets, and deployment refs are all read from the dev machine's
environment at Deploy time (ADR-0010).

| Variable | Used by | Default | Purpose |
|---|---|---|---|
| `PI_HOST`, `PI_SSH_USER` | inventory.py | `pi-zero.tailnet`, `pi` | The real Pi Zero |
| `SERVER_HOST`, `SERVER_SSH_USER` | inventory.py | `main-server.tailnet`, `admin` | The real main server |
| `PI_TEST_HOST`, `PI_TEST_SSH_PORT`, `PI_TEST_SSH_USER` | inventory.py | `localhost`, `2201`, `root` | `test` group's pi stand-in (see Testing) |
| `SERVER_TEST_HOST`, `SERVER_TEST_SSH_PORT`, `SERVER_TEST_SSH_USER` | inventory.py | `localhost`, `2202`, `root` | `test` group's server stand-in |
| `TAILSCALE_AUTH_KEY` | deploy_tailscale.py | *(required to join)* | Tailnet auth key, never committed |
| `HOMELAB_REPO_URL` | deploy_control_{pi,server}_api.py | `git@github.com:nikbogman/homelab.git` | Repo the Control APIs are pulled from |
| `DEPLOY_REF` | deploy_control_{pi,server}_api.py | `main` | Ref/commit checked out on-device |
| `SERVER_MAC_ADDRESS` | deploy_control_pi_api.py | *(required)* | WoL target MAC for the Control Pi API |
| `ALLOY_PUSH_URL` | deploy_control_{pi,server}_api.py | *(required)* | Grafana Alloy event-log endpoint |
| `CONTROL_SERVER_API_ORIGIN` | deploy_control_pi_api.py | *(required)* | Baked into the Control UI build as `VITE_CONTROL_SERVER_API_URL` |
| `CONTROL_UI_ORIGIN` | deploy_control_server_api.py | *(required)* | Control server API's CORS allow-list entry |
| `CONTROL_PI_API_HOST`/`_PORT` | deploy_control_pi_api.py, deploy_caddy.py | `127.0.0.1`, `5000` | Bind address for the Control Pi API |
| `CONTROL_SERVER_API_HOST`/`_PORT` | deploy_control_server_api.py | `127.0.0.1`, `5000` | Bind address for the Control server API |
| `MAIN_SERVER_HOST` | deploy_caddy.py | `main-server.tailnet` | Upstream host for every auto-wake-proxy route |
| `IMMICH_ROUTE_PORT`, `AGENTS_ROUTE_PORT` | deploy_caddy.py | `8443`, `8444` | Per-workload Caddy listen ports |
| `CONTROL_UI_PORT` | deploy_caddy.py | `8080` | Caddy listen port for the Control UI/Pi API site |
| `AUTO_WAKE_PROXY_BIND_HOST` | Caddyfile.j2 (Caddy env, not pyinfra) | `127.0.0.1` | Set on the device, not the dev machine |

## Running a Deploy

```sh
# Everything, both groups
uv run pyinfra inventory.py deploy.py

# Preview only -- connects and diffs, mutates nothing (see Gotcha below)
uv run pyinfra inventory.py deploy.py --dry

# One Deploy file against one Host group
uv run pyinfra inventory.py deploy_caddy.py --limit pi
uv run pyinfra inventory.py deploy_caddy.py --limit pi --dry
```

pyinfra 3.x has no `--check` flag — every "`--check`" in the ticket files
means `--dry` (`docs/agents/pyinfra.md`'s Gotchas). `deploy.py` never runs
on its own: no CI config, cron job, or git hook in this repo invokes it
(ADR-0009) -- it's a human, from the dev machine, every time.

## Testing (ticket 09)

No pytest/mypy seam applies to infrastructure code — the checks below run
through pyinfra's own execution engine, the same "highest available seam"
philosophy the spec's Testing Decisions describe. Three tiers, and this is
the precedent future Deploy files in this repo should follow:

### Tier 1 — `--dry`

Every Deploy file is run with `--dry` before being applied for real. This
connects for real and gathers facts, but executes nothing — it prints the
operations that *would* run. Catches template and logic errors early, but
per its own documented caveat it's a prepare-phase estimate, not a
correctness proof (`docs/agents/pyinfra.md`'s `--dry` section) — that's
what tiers 2 and 3 are for.

### Tier 2 — disposable containers (the `test` Host group)

`inventory.py`'s `test` group points at long-lived, **systemd-capable**
containers reached over plain `@ssh` — deliberately *not* pyinfra's
`@docker` connector. `@docker` image-mode containers run no init system
(`tail -f /dev/null` as PID 1), so the `systemd.service` operations this
feature uses throughout (tickets 02-06) can't succeed against them
(`docs/agents/pyinfra.md`'s Open Question #2). Using `@ssh` against a
systemd-capable container instead exercises the *exact same connector
path* `pi`/`server` use, so execution errors surface against the real
code path, and the container survives across runs for the idempotency
check below.

Stand up the two stand-in containers (needs a systemd-capable image with
an SSH server; adjust to taste):

```sh
docker run -d --name pi-test --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 2201:22 \
  jrei/systemd-debian:12
docker run -d --name server-test --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 2202:22 \
  jrei/systemd-debian:12

# Install your pyinfra dev machine's public key + an SSH server on each
# (exact steps depend on the chosen image; jrei/systemd-debian ships
# openssh-server disabled by default -- enable and seed authorized_keys
# via `docker exec`, then `systemctl start ssh` inside each container).
```

Then, from `pyinfra/`:

```sh
uv run pyinfra inventory.py deploy.py --limit test
```

Every Deploy file runs against both stand-ins (`device_kind` host data makes
`pi-test`/`server-test` behave like `pi`/`server` for gating purposes —
see `common.has_device_kind` and `inventory.py`'s module docstring). Bad package
names, invalid templates, and wrong command syntax surface here, against
throwaway containers, not a physical device. `deploy_tailscale.py`
deliberately skips the actual `tailscale up` join for `test` (only the
install steps run) — see that file's docstring — so this never enrolls an
ephemeral container into the real tailnet.

### Tier 3 — idempotency, the actual correctness check

Run the same Deploy twice in a row, immediately, with nothing else
changed in between, against **both** `test` and the real `pi`/`server`:

```sh
uv run pyinfra inventory.py deploy.py --limit test
uv run pyinfra inventory.py deploy.py --limit test   # again, immediately

uv run pyinfra inventory.py deploy.py --limit pi
uv run pyinfra inventory.py deploy.py --limit pi     # again, immediately
```

**Pass condition**: the second run's `Grand total` row has an empty/`-`
`Success` column and everything lands in `No Change`
(`docs/agents/pyinfra.md`'s Idempotency mechanics section has a worked
example of what that table looks like). This is what verifies a Deploy
file is genuinely declarative, not merely "ran without error" once. For a
scripted gate instead of eyeballing the table, add `--json` to the second
run and check its per-operation change counts.

### What was actually verified while building this feature

`docs/agents/pyinfra-demo.md` records the live verification performed for
pieces with no other test seam: ticket 04's shared `git_systemd_service`
helper demoed end-to-end against a real throwaway git repo and a real
(user-mode, sandbox-only) systemd unit — first run applies, second run is
fully idempotent, a code-only change correctly triggers a restart — and
ticket 07's templated Caddyfile validated with a real `caddy validate`
run. Full tiers 2 and 3 against `pi`/`server`-shaped disposable containers
require Docker and a systemd-capable image, unavailable in the sandbox
this feature was built in; the procedure above is what to run against
real infrastructure.

## Known gaps (flagged, not silently dropped)

- **`tailscale serve` port mappings** are not configured by any Deploy
  file here. Something still needs to run `tailscale serve` to actually
  expose Caddy's (and the Control server API's) loopback-bound ports to
  the tailnet with HTTPS — none of tickets 01-09 as written call this out
  as in scope, and inventing it wasn't this implementation's call to
  make. Flagged the same way the spec itself flags the bind-address
  hard-requirement gap (spec.md's Further Notes).
- **The Caddy binary itself** (built with `github.com/dulli/caddy-wol`
  via `xcaddy`, per `services/auto_wake_proxy/README.md`) is not
  installed by `deploy_caddy.py` — only the config it runs from is
  declared. Provisioning that binary is a manual prerequisite until a
  future ticket covers it (`deploy_caddy.py`'s docstring).
- **Bind-address enforcement** (the "never bound off-tailnet" hard
  requirement) is not checked by pyinfra, exactly as `spec.md` already
  notes as an explicit, deferred gap.
