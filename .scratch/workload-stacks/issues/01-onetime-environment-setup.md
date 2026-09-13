# One-time environment setup for stacks/

Status: ready-for-human

## Problem

`stacks/` deploys assume manual, one-time setup that no repo tooling performs. Nothing is usable until this exists.

## Acceptance Criteria

- [ ] `blackberry` Docker context created on each dev machine: `docker context create blackberry --docker "host=ssh://user@blackberry"`
- [ ] `docker --context blackberry ps` works and is visible from Docker Desktop / VS Code Docker extension (satisfies spec User Story 3 as a side effect — no separate tooling needed)
- [ ] `/srv/stacks/` created on blackberry
- [ ] Shared external Docker network created on blackberry (name to be picked when writing 02/03 — record the chosen name back into `docs/agents/stacks.md`)
- [ ] First `caddy-tailscale` Tailscale auth key(s) provisioned

## Notes

Manual and human-only by design (spec: no pyinfra, no repo script, box-only secrets). Blocks live deploy/verification in issue 04; does not block writing compose files in 02/03.

See spec.md Implementation Decisions ("Deploy + visibility", "Shared network", "One-time environment setup").
