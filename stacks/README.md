# Stacks

User-facing workloads running on blackberry as plain Docker Compose: outside
the Deploy, unknown to waker and hostd. Each one gets its own tsnet hostname via
the `caddy` stack; off-tailnet, edge exposes it at `/proxy/<name>/`.

Rationale: [`.scratch/workload-stacks/spec.md`](../.scratch/workload-stacks/spec.md).

## One-time setup (per dev machine, already done)

- `docker context create blackberry --docker "host=ssh://<user>@blackberry"`
- On blackberry: `/srv/stacks/` exists, the external `stacks` Docker network
  exists (`docker network create stacks`), and your SSH user is in the
  `docker` group.

## Deploying a stack

From this repo, on a machine with the `blackberry` context:

```sh
docker compose -f stacks/<name>/docker-compose.yml config                 # validate, touches nothing
docker compose --context blackberry -f stacks/<name>/docker-compose.yml up -d
docker --context blackberry ps
```

No Taskfile task, script or CI knows a stack exists.

## Adding a new stack

1. `stacks/<name>/docker-compose.yml`: plain Compose, `restart: unless-stopped`
   on every service, persistent data under `/srv/stacks/<name>/` (create it by
   hand), and join the external `stacks` network (`external: true`).
2. Secrets go in `stacks/<name>/.env`, gitignored. It must exist on the machine
   that runs `docker compose`, not on blackberry: `env_file` is resolved
   locally before the environment is sent over the ssh context.
3. To make it reachable, add a site block to [`caddy/Caddyfile`](caddy/Caddyfile):
   ```caddyfile
   :443 {
       bind tailscale/<name>
       tls {
           get_certificate tailscale
       }
       reverse_proxy <name>:<port>
   }
   ```
   No edge change is needed: edge knows no stack names.
4. Redeploy Caddy: `docker compose --context blackberry -f stacks/caddy/docker-compose.yml up -d`.

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
