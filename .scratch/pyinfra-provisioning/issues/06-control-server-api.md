# 06 — Control server API deploy file

**What to build:** `deploy_control_server_api.py` — deploys the Control server API to `server` via the shared helper (ticket 04), running as an enabled systemd service. Structurally similar to ticket 05 but a distinct application on a distinct Host group, per the repo layout decision.

**Blocked by:** 04 — needs the shared git-pull/systemd helper

**Status:** ready-for-agent

- [ ] `deploy_control_server_api.py` uses the shared helper (ticket 04) to pull the Control server API to a specific ref and run it as an enabled systemd service on `server`
- [ ] A code or unit-file change triggers a systemd restart of the Control server API as part of the Deploy
- [ ] Re-running against an already-converged `server` reports zero pending operations
