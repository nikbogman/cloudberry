# Stacks

Agent-facing reference for `stacks/<name>/` — user-facing workloads (Jellyfin, an AI agent, etc.) running on blackberry, outside pyinfra and outside the Waker/Sleeper services. Read this before touching anything under `stacks/`.

Full rationale: `.scratch/workload-stacks/spec.md`. This doc is the short-form convention reference; the spec is the "why".

## Convention

- One directory per workload: `stacks/<name>/docker-compose.yml`, plain Compose. No pyinfra Deploy file, no repo script, no CI/lint for these files — this repo's own tooling never knows a stack exists.
- Every service: `restart: unless-stopped`.
- Persistent data: `/srv/stacks/<name>/` on blackberry, created by hand per stack.
- Secrets: `stacks/<name>/.env`, created directly on blackberry over SSH, box-only, never committed (`.gitignore`'s bare `.env` pattern already covers it at any depth — no per-stack entry needed).
- Deploy and inspect from a dev machine via the `blackberry` Docker context (one-time setup, `.scratch/workload-stacks/issues/01-onetime-environment-setup.md`):
  - `docker compose --context blackberry -f stacks/<name>/docker-compose.yml up -d`
  - `docker --context blackberry ps` (also what Docker Desktop / the VS Code Docker extension use for read-only visibility)
- Validate before deploying: `docker compose -f stacks/<name>/docker-compose.yml config`.

## Shared network

All Caddy-fronted stacks join one external Docker network named **`stacks`**, created once by hand on blackberry:

```
docker network create stacks
```

Not created by any compose file — each stack's compose file declares it `external: true` and joins it.

## Routing (`stacks/caddy`)

Caddy fronts every other stack using the `caddy-tailscale` plugin (`ghcr.io/tailscale/caddy-tailscale`, verified against `/tailscale/caddy-tailscale` docs via context7), so each workload gets its own real tsnet hostname instead of a shared-hostname path prefix:

- One site block per fronted stack, `bind tailscale/<name>` + `tls { get_certificate tailscale }` — not the plain hostname as the site address, since there's no public ACME path here.
- `ephemeral false` globally (the plugin default, but stated explicitly — `ephemeral true` would drop the tsnet node's registration, and its hostname, on every container restart).
- `state_dir` under `/srv/stacks/caddy/` (mounted into the container), not the image's default XDG dirs, so tsnet state survives a container recreate.
- One Tailscale auth key per fronted site, as its own env var in Caddy's `.env` (e.g. `JELLYFIN_TS_AUTHKEY`), referenced from the matching per-node block in the Caddyfile's global `tailscale { }` options as `{$<NAME>_TS_AUTHKEY}`. Don't rely on the plugin's `TS_AUTHKEY_<NODE>` fallback — its own docs mark that path deprecated.
- Adding a new fronted stack = add its site block + per-node `auth_key` entry to `stacks/caddy/Caddyfile`, add the matching key to `stacks/caddy/.env`, join the new stack to the `stacks` network.

## Naming

Use **blackberry** (the hardware name) in Stacks docs and config, never **Sleeper** — Sleeper is the waker-service's role name for the same box and is out of the Stacks domain by design. This is a deliberate exception to the `CONTEXT.md` glossary's "avoid blackberry" rule, not an oversight.
