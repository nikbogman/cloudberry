# End-to-end deploy and verification

Status: ready-for-human

## Problem

The stacks convention isn't proven until Caddy and a fronted stack are deployed to blackberry and reachable at a real tsnet hostname. The fronted stacks are the `whoami-a` and `whoami-b` test stacks.

## Acceptance Criteria

- [ ] `docker compose --context blackberry -f stacks/whoami-a/docker-compose.yml up -d` succeeds (same for `whoami-b`)
- [ ] `docker compose --context blackberry -f stacks/caddy/docker-compose.yml up -d` succeeds
- [ ] From a tailnet device, `https://whoami-a.<tailnet>.ts.net` and `https://whoami-b.<tailnet>.ts.net` each answer with their own `WHOAMI_NAME`
- [ ] Both still answer after a suspend/wake and after a reboot of blackberry (`restart: unless-stopped`)
- [ ] Any deviation from `stacks/README.md` gets fed back into it

## Notes

Human-only: requires SSH/Tailscale access to blackberry.

Environment setup (Docker context, `/srv/stacks/`, the `stacks` network) is done against the real box. Stacks are tailnet-only: edge has no route to them.
