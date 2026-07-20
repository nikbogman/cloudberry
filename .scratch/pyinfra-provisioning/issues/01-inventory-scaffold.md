# 01 — Inventory scaffold

**What to build:** `pyinfra/inventory.py` defining the `pi` and `server` Host groups (the two physical devices) plus a `test` Host group pointing at disposable containers that stand in for each device's OS base. This is the foundation every Concern and the Deploy entrypoint targets — nothing else in this feature can be built or dry-run against a group until it exists.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] `pyinfra/inventory.py` defines a `pi` Host group matching the Pi Zero and a `server` Host group matching the main server
- [ ] A `test` Host group is defined, pointing at disposable containers standing in for the `pi` and `server` OS bases, for later execution-error testing (see ADR-adjacent Testing Decisions in the spec)
- [ ] Connecting to each group (e.g. `pyinfra inventory.py fact Os`) succeeds against real SSH connectivity for `pi`/`server` and against the containers for `test`
- [ ] Group membership and any per-host data (addresses, SSH users) live only in `inventory.py` — no Concern file hardcodes a host
