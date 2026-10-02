# Stacks

User-facing workloads (media servers, an AI agent, etc.) running on blackberry as plain Docker Compose — outside the Deploy, outside the Waker/Sleeper services. Each one gets its own tsnet hostname via the `caddy` stack; off-tailnet, Edge exposes it at `/proxy/<name>/`.

Rationale: [`.scratch/workload-stacks/spec.md`](../.scratch/workload-stacks/spec.md). Agent conventions: [`docs/stacks.md`](../docs/stacks.md).

## One-time setup (per dev machine, already done)

- `docker context create blackberry --docker "host=ssh://<user>@blackberry"`
- On blackberry: `/srv/stacks/` exists, the external `stacks` Docker network exists (`docker network create stacks`), and your SSH user is in the `docker` group.

## Deploying a stack

From this repo, on a machine with the `blackberry` context:

```
docker compose --context blackberry -f stacks/<name>/docker-compose.yml up -d
```

Check what's running:

```
docker --context blackberry ps
```

Validate a compose file before deploying (catches syntax errors without touching blackberry):

```
docker compose -f stacks/<name>/docker-compose.yml config
```

## Adding a new stack

1. `stacks/<name>/docker-compose.yml` — plain Compose, `restart: unless-stopped`, persistent data under `/srv/stacks/<name>/`, joins the external `stacks` network.
2. Secrets, if any, go in `stacks/<name>/.env` — never committed, and it must exist on whichever machine actually runs the `docker compose` command above (env vars are resolved locally, not on blackberry).
3. To make it reachable over the tailnet, add a site block to [`stacks/caddy/Caddyfile`](caddy/Caddyfile):
   ```caddyfile
   :443 {
       bind tailscale/<name>
       tls {
           get_certificate tailscale
       }
       reverse_proxy <name>:<port>
   }
   ```
   The existing `TAILSCALE_AUTH_KEY` in Caddy's `.env` covers every site — no new key needed unless a stack needs a different tailnet identity.
4. Redeploy Caddy so it picks up the new site: `docker compose --context blackberry -f stacks/caddy/docker-compose.yml up -d`.

## Where things live

| What | Where |
| --- | --- |
| Compose files | `stacks/<name>/` in this repo |
| Persistent data | `/srv/stacks/<name>/` on blackberry |
| Secrets (`.env`) | `stacks/<name>/`, box-only, gitignored |
| Shared network | external Docker network `stacks`, created once on blackberry |
| Ingress | `stacks/caddy`, one site block per fronted stack |
