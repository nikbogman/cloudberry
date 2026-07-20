# 04 — Shared git-pull + systemd deploy helper

**What to build:** A shared helper module implementing the git-pull-plus-systemd-unit pattern common to both Control APIs (repo layout decision in the spec): clone/pull this repo on the target device checked out to a specific ref, install/enable a systemd unit, and restart the service when the code or unit file changes. Parameterized per-app (ref, unit name, working directory, start command) so `deploy_control_pi_api.py` (ticket 05) and `deploy_control_server_api.py` (ticket 06) can each consume it despite being different applications on different Host groups.

**Blocked by:** 01 — needs a Host group to exercise the helper against

**Status:** ready-for-agent

- [ ] A shared helper performs: `git clone`/pull of this repo on the target device, checked out to a specific ref
- [ ] The same helper installs/enables a systemd unit and starts the service
- [ ] The helper restarts the service when a Deploy changes the pulled code or the unit file, and leaves it alone otherwise
- [ ] The helper is parameterized (ref, unit name, working directory, start command) rather than hardcoded to one app, and is demoed against a throwaway app/unit before either real Control API Concern adopts it
