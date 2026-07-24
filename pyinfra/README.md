# pyinfra provisioning

Declarative provisioning for the homelab control plane, per
`.scratch/pyinfra-provisioning/spec.md`. Converges the Pi Zero (`pi`) and
the main server (`server`) to their declared state: Tailscale joined,
Docker installed on `server`, the full Caddy config templated on `pi`, and
both the Pi API and Server API (plus the UI's static build) deployed as
systemd services. Domain vocabulary (Deploy, Deploy file, Host group) is
defined in the root [`CONTEXT.md`](../CONTEXT.md); pyinfra-specific
implementation notes and citations live in
[`docs/agents/pyinfra.md`](../docs/agents/pyinfra.md).

## Layout

```
pyinfra/
  inventory.py                    # pi / server / test Host groups (ticket 01)
  common.py                       # shared linux_codename()/linux_distro_id()/has_device_role() helpers
  settings.py                     # pydantic-settings classes -- typed env var config
  deploy_tailscale.py             # ticket 02
  deploy_docker.py                # ticket 03
  api_deploy.py                   # shared git+systemd @deploy helper (ticket 04)
  deploy_pi_api.py                # ticket 05 (+ UI static delivery)
  deploy_server_api.py            # ticket 06
  deploy_caddy.py                 # ticket 07, binary build+ship added by ADR-0013
  deploy.py                       # entrypoint composing everything (ticket 08)
  templates/
    api.service.j2                 # systemd unit template, both the Pi API and Server API
    caddy.service.j2               # systemd unit template for caddy (ADR-0013)
    Caddyfile.j2                  # the Pi's complete Caddy config
```

**Run every command from inside this directory** (`cd pyinfra/` first).
Deploy files import from sibling modules (`from common import ...`,
`from api_deploy import ...`) and `files.template` resolves
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
environment at Deploy time (ADR-0010), via typed
[`pydantic-settings`](https://docs.pydantic.dev/latest/concepts/pydantic_settings/)
classes in `settings.py` rather than each Deploy file parsing
`os.environ` by hand. Ports and similar fields are coerced to their real
type (e.g. `int`); a class with a required field (no default, e.g.
`TailscaleSettings.tailscale_auth_key`) raises a `pydantic.ValidationError`
listing every missing var at once if it isn't set. Deploy files that only
need a required setting when actually targeting a given device construct
that class lazily inside the relevant `has_device_role(...)` guard, so
running one Deploy file in isolation still never demands env vars an
unrelated one needs — the same "targetable in isolation" contract as
before. No `env_file` support — ADR-0010 already rejected a file-based
secrets store, so these classes are a typed wrapper around plain
environment variables, not a new persistence mechanism.

Copy [`secrets.sh.example`](secrets.sh.example) to `secrets.sh` (gitignored,
never committed) and fill in real values for every required variable listed
below. Run deploys via `./deploy.sh` (see Running a Deploy) rather than
`uv run pyinfra` directly — it loads `secrets.sh` into its own subprocess
and execs pyinfra, so these vars never leak into or linger in your
interactive shell.

### inventory.py (`InventorySettings`)

| Variable | Default | Purpose |
|---|---|---|
| `PI_HOST`, `PI_SSH_USER` | `pi-zero.tailnet`, `pi` | The real Pi Zero |
| `SERVER_HOST`, `SERVER_SSH_USER` | `main-server.tailnet`, `admin` | The real main server |
| `PI_TEST_HOST`, `PI_TEST_SSH_PORT`, `PI_TEST_SSH_USER` | `localhost`, `2201`, `root` | `test` group's pi stand-in (see Testing) |
| `SERVER_TEST_HOST`, `SERVER_TEST_SSH_PORT`, `SERVER_TEST_SSH_USER` | `localhost`, `2202`, `root` | `test` group's server stand-in |

### deploy_tailscale.py (`TailscaleSettings`)

| Variable | Default | Purpose |
|---|---|---|
| `TAILSCALE_AUTH_KEY` | *(required to join)* | Tailnet auth key, never committed |

### deploy_pi_api.py (`DeploySourceSettings`, `PiApiSettings`, `PiApiSecrets`)

| Variable | Default | Purpose |
|---|---|---|
| `HOMELAB_REPO_URL` | `git@github.com:nikbogman/homelab.git` | Repo the Pi API is pulled from |
| `DEPLOY_REF` | `main` | Ref/commit checked out on-device |
| `PI_API_HOST`/`_PORT` | `127.0.0.1`, `5000` | Bind address for the Pi API |
| `SERVER_MAC_ADDRESS` | *(required)* | WoL target MAC for the Pi API |
| `GRAFANA_CLOUD_LOKI_URL` | *(required)* | Grafana Cloud's Loki push endpoint -- events are POSTed here directly (ADR-0014) |
| `GRAFANA_CLOUD_LOKI_USER` | *(required)* | Grafana Cloud Loki basic-auth username (the stack's numeric instance/user ID) |
| `GRAFANA_CLOUD_LOKI_API_KEY` | *(required)* | Grafana Cloud Access Policy token, scoped to `logs:write` |
| `SERVER_API_ORIGIN` | *(required)* | Baked into the UI build as `VITE_SERVER_API_URL` |

### deploy_server_api.py (`DeploySourceSettings`, `ServerApiSettings`, `ServerApiSecrets`)

| Variable | Default | Purpose |
|---|---|---|
| `HOMELAB_REPO_URL` | `git@github.com:nikbogman/homelab.git` | Repo the Server API is pulled from |
| `DEPLOY_REF` | `main` | Ref/commit checked out on-device |
| `SERVER_API_HOST`/`_PORT` | `127.0.0.1`, `5000` | Bind address for the Server API |
| `UI_ORIGIN` | *(required)* | Server API's CORS allow-list entry |
| `GRAFANA_CLOUD_LOKI_URL` | *(required)* | Grafana Cloud's Loki push endpoint -- events are POSTed here directly (ADR-0014) |
| `GRAFANA_CLOUD_LOKI_USER` | *(required)* | Grafana Cloud Loki basic-auth username (the stack's numeric instance/user ID) |
| `GRAFANA_CLOUD_LOKI_API_KEY` | *(required)* | Grafana Cloud Access Policy token, scoped to `logs:write` |

### deploy_caddy.py (`CaddySettings`)

| Variable | Default | Purpose |
|---|---|---|
| `MAIN_SERVER_HOST` | `main-server.tailnet` | Host the `/server*` route forwards to |
| `UI_PORT` | `8080` | Caddy listen port for the single UI/Pi API/`/server*` site |
| `PI_API_PORT` | `5000` | Must match `deploy_pi_api.py`'s own `PI_API_PORT` -- read separately since each Deploy file's settings class is independent |
| `SERVER_PROXY_PORT` | *(required)* | Port on `MAIN_SERVER_HOST` that `/server*` forwards to (path stripped) -- ADR-0011; no default since the server-side proxy it points at doesn't exist yet |

### Not read by pyinfra at all

| Variable | Set where | Purpose |
|---|---|---|
| `PI_PROXY_BIND_HOST` | On the device, in Caddy's own environment | Not a dev-machine/pyinfra setting -- listed here only to avoid confusion with the pyinfra-side variables above |

## Running a Deploy

Use `./deploy.sh` in place of `uv run pyinfra inventory.py` — it loads
`secrets.sh` into its own subprocess and execs pyinfra with `inventory.py`
plus whatever args you pass, so secrets never touch your interactive shell
and you don't repeat `inventory.py` on every invocation. With no target
given it defaults to `deploy.py` (everything); otherwise it resolves a
short name to its `deploy_<name>.py` file (e.g. `caddy` → `deploy_caddy.py`)
so you don't have to keep retyping the `deploy_` prefix either — the full
filename still works if you prefer it:

```sh
# Everything, both groups
./deploy.sh

# Preview only -- connects and diffs, mutates nothing (see Gotcha below)
./deploy.sh --dry

# One Deploy file against one Host group
./deploy.sh caddy --limit pi
./deploy.sh caddy --limit pi --dry
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
./deploy.sh --limit test
```

Every Deploy file runs against both stand-ins (`device_role` host data makes
`pi-test`/`server-test` behave like `pi`/`server` for gating purposes —
see `common.has_device_role` and `inventory.py`'s module docstring). Bad package
names, invalid templates, and wrong command syntax surface here, against
throwaway containers, not a physical device. `deploy_tailscale.py`
deliberately skips the actual `tailscale up` join for `test` (only the
install steps run) — see that file's docstring — so this never enrolls an
ephemeral container into the real tailnet.

### Tier 3 — idempotency, the actual correctness check

Run the same Deploy twice in a row, immediately, with nothing else
changed in between, against **both** `test` and the real `pi`/`server`:

```sh
./deploy.sh --limit test
./deploy.sh --limit test   # again, immediately

./deploy.sh --limit pi
./deploy.sh --limit pi     # again, immediately
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

ADR-0011's `/server*` route, and ADR-0012's `call_wake_api` plugin that
replaced `caddy-wol`, were re-validated with a real `caddy validate` run
against a binary built with `services/pi-proxy/wake_plugin`
compiled in, plus a live `caddy run` smoke test against a fake Pi
API and an unreachable upstream (confirmed: the wake call fires on the
first 502, is throttled on the second, and the request still ultimately
returns a real response). That validation pass also caught a real,
pre-existing bug: `handle_errors` cannot be nested inside `handle_path`
(Caddy rejects it as "not an ordered HTTP handler") — `handle_errors` now
lives at the site's top level, matched against `orig_uri` so it still only
fires for `/server*`. This bug predates ADR-0012 and was independent of
`caddy-wol` vs. `call_wake_api`; it's fixed as part of this validation
pass since it blocked verifying the very route this change touches.

## Known gaps (flagged, not silently dropped)

- **`tailscale serve` port mappings** are not configured by any Deploy
  file here. Something still needs to run `tailscale serve` to actually
  expose Caddy's (and the Server API's) loopback-bound ports to
  the tailnet with HTTPS — none of tickets 01-09 as written call this out
  as in scope, and inventing it wasn't this implementation's call to
  make. Flagged the same way the spec itself flags the bind-address
  hard-requirement gap (spec.md's Further Notes).
- **Bind-address enforcement** (the "never bound off-tailnet" hard
  requirement) is not checked by pyinfra, exactly as `spec.md` already
  notes as an explicit, deferred gap.
- **The server-side workload proxy** that `/server*` forwards to
  (ADR-0011) doesn't exist yet — it's out of pyinfra's scope entirely
  (same boundary as the Docker Compose stacks it would front) and is a
  manual prerequisite to build, same as the Caddy binary above, before
  this route actually reaches anything.
