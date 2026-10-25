# Stacks

User-facing workloads running on blackberry as plain Docker Compose: outside
the Deploy, unknown to hostd. Each one gets its own tsnet hostname via
the `caddy` stack, reachable on the tailnet only, unless it's a
[gated stack](#gated-stacks). Rationale: [DESIGN.md](../DESIGN.md#design-decisions).

## One-time setup (per dev machine, already done)

- `docker context create blackberry --docker "host=ssh://<user>@blackberry"`
- On blackberry: `/srv/stacks/` exists, the external `stacks` Docker network
  exists (`docker network create stacks`), and your SSH user is in the
  `docker` group.

## Deploying a stack

From this repo, on a machine with the `blackberry` context:

```sh
docker compose -f stacks/<name>/docker-compose.yml config                 # validate, touches nothing
docker --context blackberry compose -f stacks/<name>/docker-compose.yml up -d
docker --context blackberry ps
```

No Taskfile task, script or CI knows a stack exists.

## Adding a new stack

1. `stacks/<name>/docker-compose.yml`: plain Compose, `restart: unless-stopped`
   on every service, persistent data under `/srv/stacks/<name>/` (create it by
   hand), and join the external `stacks` network (`external: true`).
2. Needs a database or a cache? Use the shared [`db` stack](#shared-database)
   rather than adding your own Postgres or Valkey service.
3. Secrets go in `stacks/<name>/.env`, gitignored. It must exist on the machine
   that runs `docker compose`, not on blackberry: `env_file` is resolved
   locally before the environment is sent over the ssh context.
4. To make it reachable, add a site block to [`caddy/Caddyfile`](caddy/Caddyfile):
   ```caddyfile
   :443 {
       bind tailscale/<name>
       tls {
           get_certificate tailscale
       }
       reverse_proxy <name>:<port>
   }
   ```
5. Redeploy Caddy: `docker --context blackberry compose -f stacks/caddy/docker-compose.yml up -d`.

## Caddy

Uses the `caddy-tailscale` plugin (`ghcr.io/tailscale/caddy-tailscale`), so
each stack is its own tailnet node rather than a path on a shared hostname.

- Site addresses are `bind tailscale/<name>` with `get_certificate tailscale`,
  not a plain hostname: there's no public ACME path.
- `ephemeral false` is set explicitly. `true` would drop each node, and its
  hostname, on every container restart.
- `state_dir` is under `/srv/stacks/caddy/` (mounted), so tsnet state survives
  a container recreate.
- One global `auth_key {$TAILSCALE_AUTH_KEY}` covers every site. A site that
  needs a different tailnet identity or tags gets its own key in a per-node
  block; the plugin's `TS_AUTHKEY_<NODE>` fallback is deprecated.
- Caddy's `.env` is at the repo root (`env_file: ../../.env`), not
  `stacks/caddy/.env`. Move it when convenient.

## Gated stacks

A stack listed in [`cmd/gateway/routes.json`](../cmd/gateway/routes.json) is reached
through [gateway](../cmd/gateway/README.md) on raspberry, which wakes blackberry on
request. It skips Caddy: gateway terminates TLS, so the stack serves plain HTTP
on a port published on blackberry's static LAN IP (never `0.0.0.0`, and not the
tailnet IP, which may not exist yet when Docker starts). A firewall rule on
blackberry lets only raspberry reach that port; see
[DESIGN.md](../DESIGN.md#security). Examples:
[`immich/`](immich/) (own port 8443), [`popcorn/`](#popcorn) (path `/popcorn`).

1. Compose and `.env` as above, plus the published port bound to the LAN IP.
2. Add a route to `routes.json` with `"upstream": "http://<lan-ip>:<port>"`.
   It gets its own HTTPS port on raspberry (`"port": 8444`), or, for an API
   that works behind a prefix, a path on 443 (`"path": "/<name>"`, with
   `port` as the loopback port only). Web UIs like Immich need their own port.
   See [DESIGN.md](../DESIGN.md#design-decisions).
3. Add the port to `PORTS` in
   [`stack-firewall.service`](../deployments/blackberry/stack-firewall.service),
   then `task firewall`: only raspberry may reach it.
4. `task gateway` to ship the routes.

## Shared database

[`db/`](db/) runs the only Postgres and Valkey on blackberry. Stacks join its
`db` network (`external: true`) and connect to `postgres:5432` and
`valkey:6379`; nothing is published. Postgres is the official 17 image plus
pgvector. Start it before any stack that uses it.
Data is under `/srv/stacks/db/` on the SSD. Rationale:
[DESIGN.md](../DESIGN.md#design-decisions).

Giving a stack its own database (blackberry awake):

```sh
stacks/db/new-database.sh <name>
```

It creates role and database `<name>` with a generated password, and appends
`DB_PASSWORD`, `DATABASE_URL` (host `postgres`, for containers on the `db`
network) and `LOCAL_DATABASE_URL` (`127.0.0.1`, through the tunnel below) to
`stacks/<name>/.env`. The password is never printed.

From the dev machine, for occasional scripts, tunnel to the container (WSL
forwards it to Windows' `127.0.0.1` too):

```sh
IP=$(docker --context blackberry inspect -f '{{(index .NetworkSettings.Networks "db").IPAddress}}' db-postgres-1)
ssh -N -L 5432:$IP:5432 blackberry
``` Only Immich's role is a superuser,
because Immich requires it.

Valkey has no users here: each stack gets its own DB index instead. Taken:

| Index | Stack |
|---|---|
| 0 | immich |

## Popcorn

[`popcorn/`](popcorn/) is the API for the Popcorn mobile app
([nikbogman/popcorn](https://github.com/nikbogman/popcorn), `server/`). It
is gated at `https://raspberry.<tailnet>.ts.net/popcorn`, so sharing a reel from
the phone wakes blackberry. The app polls the job while it runs, which keeps
gateway's idle timer from firing. Tailnet-only: the phone needs Tailscale on
(Android's "Always-on VPN").

The image is `ghcr.io/nikbogman/popcorn-server:latest` (public), built by the
popcorn repo's GitHub Action on pushes to `main`. The compose file sets
`pull_policy: always`, so a plain `up -d` deploys the newest one. Wait for the
Action to finish first. To roll back, set `image:` to a `sha-<short>` tag.

Once:

1. `stacks/db/new-database.sh popcorn` (done). It writes `DATABASE_URL` to
   `popcorn/.env`.
2. Add `TMDB_API_KEY`, `GEMINI_API_KEY` and `POPCORN_BIND_IP` (blackberry's
   LAN IP) to `popcorn/.env`. For `LOGIN_PASSWORD`, run
   `uv run python -m app.password <cloudberry>/stacks/popcorn/.env` in the
   popcorn checkout's `server/`. It writes a random one without printing it.
   Optional: `BROWSER_IDLE_MINUTES` (default 5) before Chromium is closed.
3. `task firewall` (port 3000 is in `PORTS`), then `task gateway` (route `/popcorn`).
4. Instagram login. The session is stored in the database, so run the login on
   the dev machine through the [tunnel](#shared-database), never on blackberry:
   ```sh
   cd <popcorn checkout>/server
   DATABASE_URL=<LOCAL_DATABASE_URL from popcorn/.env> uv run python -m app.reels.login
   ```
   Repeat when reel jobs fail with the login hint.
5. In the app's EAS `preview` environment, set
   `EXPO_PUBLIC_API_URL=https://raspberry.<tailnet>.ts.net/popcorn`.

Deploy or update:

```sh
docker --context blackberry compose -f stacks/popcorn/docker-compose.yml up -d
```
