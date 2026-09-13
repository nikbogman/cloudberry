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

## Comments

2026-09-13: Issue 02 (Jellyfin) was implemented then dropped by user request — see its Comments. Acceptance criteria here still reference Jellyfin as the fronted stack; re-target them at whatever stack actually gets deployed once one exists (issue 05's AI agent workload, once scoped, or another). Issue 01's environment setup and issue 03's Caddy stack (now with no site blocks) are otherwise complete against the real box.
