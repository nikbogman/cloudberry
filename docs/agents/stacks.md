# Stacks

Agent-facing reference for `stacks/<name>/` — user-facing workloads running on blackberry, outside pyinfra and outside the Waker/Sleeper services. Read this before touching anything under `stacks/`.

Full rationale: `.scratch/workload-stacks/spec.md`. This doc is the short-form convention reference; the spec is the "why".

## Convention

- One directory per workload: `stacks/<name>/docker-compose.yml`, plain Compose. No pyinfra Deploy file, no repo script, no CI/lint for these files — this repo's own tooling never knows a stack exists.
- Every service: `restart: unless-stopped`.
- Persistent data: `/srv/stacks/<name>/` on blackberry, created by hand per stack.
- Secrets: `stacks/<name>/.env`, box-only, never committed (`.gitignore`'s bare `.env` pattern already covers it at any depth — no per-stack entry needed). Correction to the spec's original assumption: `.env` must exist wherever `docker compose` is actually *invoked* (the dev machine's repo checkout), not on blackberry's filesystem — `docker compose --context blackberry` resolves `env_file` locally before sending the container's environment to blackberry's daemon over the SSH context, it never reads anything from blackberry's disk.
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
- One Tailscale auth key, `TAILSCALE_AUTH_KEY` in Caddy's `.env`, referenced once as the global `auth_key {$TAILSCALE_AUTH_KEY}` in the Caddyfile's `tailscale { }` options — applies to every node this Caddy instance registers. Move to a per-site key (its own env var, set in a per-node block inside `tailscale { }`) if a future site needs to authenticate as a different tailnet identity or use different tags; don't rely on the plugin's `TS_AUTHKEY_<NODE>` fallback for that, its own docs mark that path deprecated.
- Adding a new fronted stack = add its site block to `stacks/caddy/Caddyfile` (`bind tailscale/<name>`), join the new stack to the `stacks` network. The existing `auth_key` covers it automatically.
- Off-tailnet, Edge (`waker-service/docs/edge.md`) exposes every site as `/proxy/<name>/*` → `https://<name>.<tailnet>.ts.net/*`. It knows no stack names, so a new site block needs no Edge change.
- Deviation from the `stacks/<name>/.env` convention above: Caddy's `.env` currently lives at the repo root, and `stacks/caddy/docker-compose.yml`'s `env_file` points at it via `../../.env`. Move it to `stacks/caddy/.env` and drop the relative path whenever convenient — not urgent, since either location is equally untracked and equally readable by whoever runs the deploy.

## Naming

Use **blackberry** (the hardware name) in Stacks docs and config, never **Sleeper** — Sleeper is the waker-service's role name for the same box and is out of the Stacks domain by design. This is a deliberate exception to the `CONTEXT.md` glossary's "avoid blackberry" rule, not an oversight.
