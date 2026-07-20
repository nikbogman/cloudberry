# 05 — Control Pi API Concern (+ Control UI static delivery)

**What to build:** `deploy_control_pi_api.py` — deploys the Control Pi API to `pi` via the shared helper (ticket 04), running as an enabled systemd service. Also delivers the Control UI: built on the dev machine (Vite build, per ADR-0006) and only the static `dist/` output copied to `pi` — the Pi never runs npm/Node.

**Blocked by:** 04 — needs the shared git-pull/systemd helper

**Status:** ready-for-agent

- [ ] `deploy_control_pi_api.py` uses the shared helper (ticket 04) to pull the Control Pi API to a specific ref and run it as an enabled systemd service on `pi`
- [ ] A code or unit-file change triggers a systemd restart of the Control Pi API as part of the Deploy
- [ ] The Control UI is built on the dev machine and only its static build output is delivered to `pi` — no Node/npm toolchain ever runs on the device
- [ ] Re-running against an already-converged `pi` reports zero pending operations
