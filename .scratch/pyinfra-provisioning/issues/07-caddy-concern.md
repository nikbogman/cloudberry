# 07 — Caddy Concern

**What to build:** `deploy_caddy.py` — templates the entire Caddyfile on `pi` from a declarative source in this repo (ADR-0008), including the auto-wake-proxy routes for every current workload service (Immich, AI agents), the route(s) serving the Control UI's static output, and the route to the Control server API. No hand-edited Caddy config is ever left on the device.

**Blocked by:** 02 (tailnet must be joined before Caddy binds tailnet-only addresses), 05 (needs the Control UI static path to route to), 06 (needs the Control server API to route to)

**Status:** ready-for-agent

- [ ] `deploy_caddy.py` templates the complete Caddyfile on `pi` from a single declarative source in this repo — nothing is hand-edited on the device
- [ ] The templated Caddyfile includes the auto-wake-proxy route for every current workload service (Immich, AI agents)
- [ ] The templated Caddyfile routes to the Control UI's static build output and to the Control server API
- [ ] Adding a new workload service's route requires only editing this repo and redeploying — never a direct edit on `pi`
- [ ] The Concern is targetable/dry-runnable in isolation against `pi`
- [ ] Re-running against an already-converged `pi` reports zero pending operations
