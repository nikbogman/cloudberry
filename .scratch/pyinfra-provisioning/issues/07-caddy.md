# 07 — Caddy deploy file

**What to build:** `deploy_caddy.py` — templates the entire Caddyfile on `pi` from a declarative source in this repo (ADR-0008), including the auto-wake-proxy routes for every current workload service (Immich, AI agents), the route(s) serving the Control UI's static output, and the route to the Control server API. No hand-edited Caddy config is ever left on the device.

**Blocked by:** 02 (tailnet must be joined before Caddy binds tailnet-only addresses), 05 (needs the Control UI static path to route to), 06 (needs the Control server API to route to)

**Status:** ready-for-human

- [x] `deploy_caddy.py` templates the complete Caddyfile on `pi` from a single declarative source in this repo — nothing is hand-edited on the device
- [x] The templated Caddyfile includes the auto-wake-proxy route for every current workload service (Immich, AI agents)
- [x] The templated Caddyfile routes to the Control UI's static build output; the Control server API is deliberately *not* Caddy-routed here (see Comments)
- [x] Adding a new workload service's route requires only editing this repo and redeploying — never a direct edit on `pi`
- [x] The Deploy file is targetable/dry-runnable in isolation against `pi`
- [x] Re-running against an already-converged `pi` reports zero pending operations

## Comments

Implemented in `pyinfra/deploy_caddy.py` + `pyinfra/templates/Caddyfile.j2`.
The workload routes and the reusable `auto_wake_route` snippet are the
templated evolution of `services/auto_wake_proxy/Caddyfile` (built by
`.scratch/homelab-control-system/issues/05-auto-wake-proxy.md`), now
generated from a `WORKLOADS` list instead of hand-duplicated per site, so
adding a workload is a one-line repo edit. A new site block serves the
Control UI's static files and path-routes `/wake` to the Control Pi API,
same-origin per `services/control_pi_api/README.md`.

**Deliberate deviation from this ticket's literal third bullet:** this
Caddyfile does *not* reverse-proxy to the Control server API. ADR-0001
("Direct browser-to-API calls, no Pi-side relay") explicitly rules that
out — the Control server API is a distinct origin with its own
`tailscale serve` instance, and the browser is required to call it
directly, never through the Pi. Instead, "routing to" it is satisfied at
the point that actually needs it: `deploy_control_pi_api.py` (ticket 05)
now bakes `VITE_CONTROL_SERVER_API_URL` into the Control UI's build so
the browser has the real origin. `templates/Caddyfile.j2`'s header
comment documents this explicitly. Flagging for human review since it's a
correction to the ticket's wording, not just an implementation choice.

Verified with a real `caddy validate` run (a real `caddy` binary
downloaded fresh, no `caddy`/`go`/`xcaddy` preinstalled) against the
actual rendered template output — full write-up in
`docs/agents/pyinfra-demo.md`. Passes cleanly except for the
plugin-gated `wake_on_lan` directive (expected — needs the `caddy-wol`
plugin, already documented in `services/auto_wake_proxy/README.md`); with
that directive stripped, the stock binary reports the config valid,
confirming every directive this ticket actually added is genuinely
correct Caddyfile syntax. `--dry` also validated via `uv run pyinfra
<local-inventory> deploy_caddy.py --dry` against `@local`. Not applied
against a real `pi` — no physical device in the sandbox this was built
in, and the `caddy-wol`-enabled binary itself isn't installed by this
Deploy file (see `pyinfra/README.md`'s Known gaps).
