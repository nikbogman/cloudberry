# 05 — Control Pi API deploy file (+ Control UI static delivery)

**What to build:** `deploy_control_pi_api.py` — deploys the Control Pi API to `pi` via the shared helper (ticket 04), running as an enabled systemd service. Also delivers the Control UI: built on the dev machine (Vite build, per ADR-0006) and only the static `dist/` output copied to `pi` — the Pi never runs npm/Node.

**Blocked by:** 04 — needs the shared git-pull/systemd helper

**Status:** ready-for-human

- [x] `deploy_control_pi_api.py` uses the shared helper (ticket 04) to pull the Control Pi API to a specific ref and run it as an enabled systemd service on `pi`
- [x] A code or unit-file change triggers a systemd restart of the Control Pi API as part of the Deploy
- [x] The Control UI is built on the dev machine and only its static build output is delivered to `pi` — no Node/npm toolchain ever runs on the device
- [x] Re-running against an already-converged `pi` reports zero pending operations

## Comments

Implemented in `pyinfra/deploy_control_pi_api.py`: `git_systemd_service`
(ticket 04) targets `dest=/srv/homelab` (the monorepo clone) with
`working_directory=/srv/homelab/services/control_pi_api` — the helper
gained a `working_directory` parameter, distinct from the git clone
`dest`, specifically for this monorepo-subdirectory case. The Control UI
build uses `pyinfra.local.shell` (runs on the dev machine, not the
target) followed by `files.sync` of `dist/` to `pi`; `VITE_CONTROL_SERVER_API_URL`
is baked in at build time from `CONTROL_SERVER_API_ORIGIN` so the browser
calls the Control server API directly (ADR-0001 — no relay through this
Pi's Caddy; see `deploy_caddy.py`'s docstring for why ticket 07's "routes
to the Control server API" bullet is satisfied here, not as a Caddy
reverse-proxy).

Verified via `uv run pyinfra <local-inventory> deploy_control_pi_api.py --dry`
against `@local`: the Control UI build step is **not simulated** — `npm
ci && npm run build` genuinely ran and produced a real `services/control_ui/dist`
(gitignored) during this validation. The remote (git-pull/systemd/sync)
side was diff-previewed only, not applied — no physical `pi` available in
the sandbox this was built in.
