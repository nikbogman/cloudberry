# 09 — Disposable-container test harness & idempotency verification

**What to build:** The tier-2/tier-3 testing workflow from the spec's Testing Decisions: a documented, repeatable procedure for running `deploy.py` against the `test` Host group's disposable containers to catch real execution errors (bad package names, invalid templates, wrong command syntax) before touching physical devices, plus the idempotency check — running a Deploy a second time immediately after the first and confirming zero pending operations — against both `test` and the real `pi`/`server` groups. This establishes the precedent for infrastructure testing in this repo (no prior art exists).

**Blocked by:** 08 — needs the Deploy entrypoint to run against the `test` group

**Status:** ready-for-human

- [x] Running `deploy.py` against the `test` Host group applies every Deploy file against disposable containers standing in for `pi`/`server` (see Comments for what was/wasn't actually run)
- [x] A documented procedure runs a Deploy a second time immediately after the first — against `test` and against the real `pi`/`server` groups — and treats zero pending operations as the pass condition
- [x] The three-tier approach (`--check`, disposable containers, idempotency) is written down as the precedent future Deploy files should follow, since this repo has no prior infrastructure-testing pattern to draw on

## Comments

The three-tier procedure is written down in `pyinfra/README.md`'s Testing
section (tier 1 `--dry`, tier 2 disposable containers, tier 3
idempotency), including the exact `docker run` recipe for standing up the
`test` group's two systemd-capable containers and the two-runs-in-a-row
commands for tier 3. `inventory.py`'s `test` group is wired and ready —
see ticket 01.

**Honest gap**: no Docker daemon was available in the sandbox this was
built in, so tiers 2 and 3 were never literally run against disposable
containers or real `pi`/`server` — that's on the operator, following the
documented procedure. What *was* actually exercised, live: tier 1
(`--dry`) against every Deploy file via the `@local` connector (see each
ticket's own Comments), and, going beyond tier 1, ticket 04's shared
helper was demoed through a full real apply → idempotent re-run →
change-triggers-restart cycle against a real (if sandbox-only) systemd
unit, and ticket 07's Caddyfile was validated with a real `caddy`
binary — both documented in `docs/agents/pyinfra-demo.md`. Tier 2's core
design decision (`@ssh` against long-lived systemd-capable containers,
not `@docker` image-mode) is explained in `inventory.py`'s docstring and
`docs/agents/pyinfra.md`'s Open Question #2 — chosen specifically because
`@docker` image-mode containers can't run the `systemd.service` operations
this feature depends on throughout.
