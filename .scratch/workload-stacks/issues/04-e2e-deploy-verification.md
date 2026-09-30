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

Human-only: requires SSH/Tailscale access to blackberry that isn't available to an AFK agent.

Environment setup (Docker context, `/srv/stacks/`, shared network) and the Caddy stack are both done against the real box. Jellyfin was implemented as the first proof of the `stacks/<name>/` convention, then dropped by user request before ever being deployed — Caddy currently has no site blocks and fronts nothing.

## Comments

2026-09-13: Acceptance criteria above still reference Jellyfin as the fronted stack; re-target them at whatever stack actually gets deployed once one exists (issue 05's AI agent workload, once scoped, or another).
