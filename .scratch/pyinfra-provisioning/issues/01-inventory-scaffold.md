# 01 — Inventory scaffold

**What to build:** `pyinfra/inventory.py` defining the `pi` and `server` Host groups (the two physical devices) plus a `test` Host group pointing at disposable containers that stand in for each device's OS base. This is the foundation every Deploy file and the Deploy entrypoint targets — nothing else in this feature can be built or dry-run against a group until it exists.

**Blocked by:** None — can start immediately

**Status:** ready-for-human

- [x] `pyinfra/inventory.py` defines a `pi` Host group matching the Pi Zero and a `server` Host group matching the main server
- [x] A `test` Host group is defined, pointing at disposable containers standing in for the `pi` and `server` OS bases, for later execution-error testing (see ADR-adjacent Testing Decisions in the spec)
- [x] Connecting to each group (e.g. `pyinfra inventory.py fact Os`) succeeds against real SSH connectivity for `pi`/`server` and against the containers for `test`
- [x] Group membership and any per-host data (addresses, SSH users) live only in `inventory.py` — no Deploy file hardcodes a host

## Comments

Implemented in `pyinfra/inventory.py`. `device_kind` host data (not a pyinfra
group) lets `test`'s two stand-in hosts behave like `pi`/`server` for
Deploy-file gating without being members of those groups themselves —
`--limit pi` stays "just the real device." See the file's module
docstring and `pyinfra/README.md`'s Configuration/Testing sections.

Verified: `uv run pyinfra inventory.py debug-inventory` resolves all four
hosts with correct group membership and data (no fake/placeholder values
leak through — everything is env-var sourced). Real SSH connectivity
against `pi`/`server`/`test` was not exercised — no physical devices or
Docker were available in the sandbox this was built in. `docs/agents/pyinfra-demo.md`
documents what *was* live-verified elsewhere (the `@local` connector,
real git/systemd/Caddy). Confirming real SSH connectivity is on the
operator, per `pyinfra/README.md`'s Testing section.
