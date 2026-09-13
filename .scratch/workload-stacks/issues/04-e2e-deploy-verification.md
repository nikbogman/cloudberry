# End-to-end deploy and verification

Status: ready-for-human

## Problem

Nothing in this spec is proven until Jellyfin + Caddy are actually deployed to blackberry and reachable at a real tsnet hostname. This is the live check the spec's Testing Decisions call for.

## Acceptance Criteria

- [ ] `docker compose --context blackberry -f stacks/jellyfin/docker-compose.yml up -d` succeeds
- [ ] `docker compose --context blackberry -f stacks/caddy/docker-compose.yml up -d` succeeds
- [ ] Jellyfin answers at its tsnet hostname (e.g. `jellyfin.<tailnet>.ts.net`) through Caddy
- [ ] Restart survives a reboot/suspend-wake of blackberry (spec User Story 6) — reboot and re-check
- [ ] Any deviation found from `docs/agents/stacks.md` or `docs/adr/DECISIONS.md` gets fed back into those docs

## Notes

Human-only: requires SSH/Tailscale access to blackberry that isn't available to an AFK agent. Blocked by issues 01, 02, 03.
