# 04 — Shared git-pull + systemd deploy helper

**What to build:** A shared helper module implementing the git-pull-plus-systemd-unit pattern common to both Control APIs (repo layout decision in the spec): clone/pull this repo on the target device checked out to a specific ref, install/enable a systemd unit, and restart the service when the code or unit file changes. Parameterized per-app (ref, unit name, working directory, start command) so `deploy_control_pi_api.py` (ticket 05) and `deploy_control_server_api.py` (ticket 06) can each consume it despite being different applications on different Host groups.

**Blocked by:** 01 — needs a Host group to exercise the helper against

**Status:** ready-for-human

- [x] A shared helper performs: `git clone`/pull of this repo on the target device, checked out to a specific ref
- [x] The same helper installs/enables a systemd unit and starts the service
- [x] The helper restarts the service when a Deploy changes the pulled code or the unit file, and leaves it alone otherwise
- [x] The helper is parameterized (ref, unit name, working directory, start command) rather than hardcoded to one app, and is demoed against a throwaway app/unit before either real Control API Deploy file adopts it

## Comments

Implemented as `git_systemd_service` in `pyinfra/control_api_deploy.py`
(an `@deploy`-decorated function), rendering
`pyinfra/templates/control-api.service.j2`. Restart/reload gating uses
each operation's `.will_change` property, not `.did_change()` — the
latter is a method that raises until the operation has actually executed,
so it can't be read synchronously in the same prepare-phase pass; see the
file's docstring and `docs/agents/pyinfra-demo.md`'s closing note.

**Demoed for real**, per this ticket's explicit requirement, against a
throwaway git repo and a real (user-mode, sandbox-only) systemd unit —
full write-up in `docs/agents/pyinfra-demo.md`. Three live runs: first
applies for real (service genuinely running, verified via `systemctl
status` and a heartbeat file the demo app wrote), second is fully
idempotent (`Grand total: 3 success=-, no_change=3`), third — a
code-only change with the unit file untouched — correctly restarts
(`will_change` gating proven, not just plausible). Cleaned up afterward;
nothing from the demo is part of this repo.
