# Caddy stack with caddy-tailscale

Status: ready-for-agent

## Problem

There's no routing story since the old Docker-aware reverse proxy was removed. Each workload needs its own real tsnet hostname via the `caddy-tailscale` plugin, not a shared-hostname path prefix.

## Acceptance Criteria

- [ ] `stacks/caddy/docker-compose.yml`: plain Compose, `restart: unless-stopped`, `state_dir` under `/srv/stacks/caddy/`
- [ ] Caddyfile binds each site with `bind tailscale/<name>`, `ephemeral: false`, one site per stack it fronts (Jellyfin first)
- [ ] Each site's Tailscale auth key lives in Caddy's own `.env` (created on blackberry, not committed)
- [ ] Joins the shared external Docker network from issue 01
- [ ] Config verified against `caddy-tailscale` docs via context7 (`/tailscale/caddy-tailscale`) per the rule in `~/.claude/rules/context7.md` — re-verify rather than trusting the spec's earlier pass
- [ ] `docker compose -f stacks/caddy/docker-compose.yml config` validates cleanly

## Out of Scope

- Live verification that a hostname actually resolves and serves traffic — that's issue 04, blocked on issue 01's Tailscale auth key provisioning.

## Notes

Depends on issue 02 existing (fronts Jellyfin) for a concrete site to configure, though the Caddy compose/Caddyfile skeleton can be written in parallel.

See spec.md User Stories 7, 9 and Implementation Decisions ("Routing").
