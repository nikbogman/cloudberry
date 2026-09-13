# Jellyfin stack

Status: wontfix

## Problem

Jellyfin is the first proof of the `stacks/<name>/` convention — simplest workload, no GPU/model complexity — and needs to exist before Caddy routing can be proven against it.

## Acceptance Criteria

- [ ] `stacks/jellyfin/docker-compose.yml`: plain Compose, `restart: unless-stopped`, persistent data under `/srv/stacks/jellyfin/`
- [ ] Joins the shared external Docker network from issue 01 (network is external/pre-existing, not created by this compose file)
- [ ] Secrets, if any, via `stacks/jellyfin/.env` (created on blackberry, not committed — do not add a `.env` to this repo, `.gitignore` should already cover `stacks/*/.env`)
- [ ] `docker compose -f stacks/jellyfin/docker-compose.yml config` validates cleanly
- [ ] Follows `docs/agents/stacks.md` conventions (read it first)

## Out of Scope

- Deploying to blackberry or verifying the tsnet hostname resolves — that's issue 04, and needs issue 01 done first.
- Any Caddy/routing config — that's issue 03.

## Notes

See spec.md User Stories 2, 4, 5, 6, 8 and Implementation Decisions ("Scope", "Persistence", "Restart behavior").

## Comments

2026-09-13: Implemented, deployed against real infra (blackberry context/network/directories from issue 01), then dropped by user request — Jellyfin is no longer a feature of this stacks effort. `stacks/jellyfin/` removed; `stacks/caddy/Caddyfile`'s Jellyfin site block removed. The `stacks/<name>/` convention itself (proved via this ticket before the drop) stands; see `docs/agents/stacks.md`.
