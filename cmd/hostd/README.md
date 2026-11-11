# hostd

blackberry's own daemon. Exposes a Reachable health check and the Suspend
action, and deploys the Stacks: it keeps blackberry in step with the
[deploys](https://github.com/nikbogman/deploys) repo. Fronted by its own
`tailscale serve` instance; called by the [UI](../../ui/README.md), straight
from the browser, and by gateway (auto-suspend, routes).

Vocabulary (blackberry, Reachable, Suspend) is in [`CONTEXT.md`](../../CONTEXT.md).

Code: [`internal/hostd/`](../../internal/hostd),
entrypoint [`cmd/hostd/`](.).

## Routes

| Route | Behavior |
|---|---|
| `GET /health` | Requires the identity header. Returns `{"reachable": true}`. Logs `reachability_changed` the first time; it can never observe going unreachable, since the process is asleep whenever that's true. |
| `GET /routes` | Requires the identity header. gateway's routes file, built from the synced stacks' `x-route` blocks; see [gateway's routes file](../gateway/README.md#routes-file). 404 without `STACKS_DIR`. |
| `POST /suspend` | Requires the identity header. Runs `systemctl suspend` (see [`suspend.go`](../../internal/hostd/suspend.go)), logs `suspend_requested`/`_succeeded`/`_failed`, returns `{"suspend": ...}`. |

The suspend command isn't configurable via env var, so misconfiguration can't
reintroduce a full shutdown.

### Suspend never waits for in-flight work

Userspace freezes mid-request: no FIN, no 503, just silence. Clients stall
until their own timeout; on resume the handler finishes the work and answers a
socket nobody is reading. A non-idempotent request can therefore run twice if
the client retries after the next Wake. Possible fixes:
[`.scratch/suspend-inflight-work.md`](../../.scratch/suspend-inflight-work.md).

## Stack sync

Every `$UPDATE_MINUTES` while awake, hostd runs `git fetch` in its clone of
deploys (`$STACKS_DIR`). If `main` moved past the `deployed` branch, the last
applied commit, it:

1. `git reset --hard origin/main`. The repo is the source of truth; hand edits
   in the clone are discarded.
2. Decrypts each `<stack>/secrets.env` to `<stack>/.env` (mode 600) with sops,
   using `SOPS_AGE_KEY_FILE`.
3. For each top-level folder the diff touched that has (or had) a
   `compose.yml`, in name order: `docker compose up -d --remove-orphans`, or
   `docker compose -p <folder> down` if it's gone (and its `.env` is deleted).
   A stack whose secrets didn't decrypt is skipped, never started.
4. Moves `deployed` to the new commit, even if a stack failed: it's retried on
   the next push, not every poll.
5. Writes the routed stacks' ports to stack-firewall's drop-in
   (`$FIREWALL_DROPIN`) and restarts it, if they changed.

With no `deployed` branch (a fresh clone), the first sync only records `main`
as deployed (`stacks_adopted`): starting hostd never restarts a running stack
or starts one left down. Events: `stack_deployed`, `stack_removed`,
`stack_skipped`, `stack_failed`, `firewall_updated`, and `stacks_sync_failed`
(once per distinct error, e.g. GitHub unreachable).

hostd decides from the repo alone and never reads Docker's state. The clone,
the deploy key and the age key come from `task stacks`; see
[`DEPLOY.md`](../../DEPLOY.md).

## Updates

After each sync, hostd checks for a new release and swaps itself to it like
gateway does: release builds only, `.prev` kept, three failed starts in a
minute roll back to it. See [gateway's updates](../gateway/README.md#updates).

## Environment

Read by [`internal/hostd/config.go`](../../internal/hostd/config.go), the source
of truth for names and defaults. Set in the repo root `.env` on the dev
machine; `task hostd` passes them to the unit. Bold variables are required.

| Variable | Purpose |
|---|---|
| **`UI_ORIGIN`** | The UI's Cloudflare Pages origin, allowed by CORS. Unset disables CORS (required by the deploy). |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `HOSTD_PORT` | Listen port (default `5000`). The deploy pins it, since `tailscale serve` must point at it. |
| `STACKS_DIR` | The deploys clone to sync (the deploy sets `/srv/deploys`). Unset turns the sync and `GET /routes` off. |
| `SOPS_AGE_KEY_FILE` | Read by sops: blackberry's age key (the deploy sets `/etc/hostd/age.key`). |
| `FIREWALL_DROPIN` | stack-firewall's `PORTS` override (default `/etc/systemd/system/stack-firewall.service.d/ports.conf`). |
| `UPDATE_MINUTES` | How often to sync and check for a release (default `2`). The clock stops in suspend, so the first check after a wake can be up to one interval late. |
| `RELEASE_URL` | Where releases come from (default `https://github.com/nikbogman/metal`). Empty turns binary updates off. |
| `HOSTD_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet. The deploy never sets it. |

## Deployment

`task hostd` builds for blackberry's architecture (read over ssh), ships the
binary, a systemd unit and its rollback unit, and runs `tailscale serve` for
`$HOSTD_PORT`. It's
a systemd unit rather than a container so it stays controllable while Docker
itself redeploys. See [`DEPLOY.md`](../../DEPLOY.md).
