# 06 — Control server API deploy file

**What to build:** `deploy_control_server_api.py` — deploys the Control server API to `server` via the shared helper (ticket 04), running as an enabled systemd service. Structurally similar to ticket 05 but a distinct application on a distinct Host group, per the repo layout decision.

**Blocked by:** 04 — needs the shared git-pull/systemd helper

**Status:** ready-for-human

- [x] `deploy_control_server_api.py` uses the shared helper (ticket 04) to pull the Control server API to a specific ref and run it as an enabled systemd service on `server`
- [x] A code or unit-file change triggers a systemd restart of the Control server API as part of the Deploy
- [x] Re-running against an already-converged `server` reports zero pending operations

## Comments

Implemented in `pyinfra/deploy_control_server_api.py`, mirroring ticket
05's structure: `git_systemd_service` targets
`working_directory=/srv/homelab/services/control_server_api` under the
same monorepo clone. Gated on `common.has_device_kind("server")`.

Verified via `uv run pyinfra <local-inventory> deploy_control_server_api.py --dry`
against `@local` (device_kind=server: full operation set; device_kind=pi: correctly a
no-op). Idempotent re-run against a real `server` not verified — no
physical hardware in the sandbox this was built in.
