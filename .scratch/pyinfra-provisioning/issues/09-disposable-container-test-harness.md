# 09 — Disposable-container test harness & idempotency verification

**What to build:** The tier-2/tier-3 testing workflow from the spec's Testing Decisions: a documented, repeatable procedure for running `deploy.py` against the `test` Host group's disposable containers to catch real execution errors (bad package names, invalid templates, wrong command syntax) before touching physical devices, plus the idempotency check — running a Deploy a second time immediately after the first and confirming zero pending operations — against both `test` and the real `pi`/`server` groups. This establishes the precedent for infrastructure testing in this repo (no prior art exists).

**Blocked by:** 08 — needs the Deploy entrypoint to run against the `test` group

**Status:** ready-for-agent

- [ ] Running `deploy.py` against the `test` Host group applies every Concern against disposable containers standing in for `pi`/`server`, and real execution errors (bad package names, invalid templates, wrong command syntax) surface here rather than on a physical device
- [ ] A documented procedure runs a Deploy a second time immediately after the first — against `test` and against the real `pi`/`server` groups — and treats zero pending operations as the pass condition
- [ ] The three-tier approach (`--check`, disposable containers, idempotency) is written down as the precedent future Concerns should follow, since this repo has no prior infrastructure-testing pattern to draw on
