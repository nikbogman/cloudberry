# 03 — Docker deploy file

**What to build:** `deploy_docker.py` — a single-responsibility Deploy file that installs and enables the Docker engine on the `server` Host group, so the (out-of-scope) workload Compose stacks have an engine to run on. Targetable and dry-runnable in isolation.

**Blocked by:** 01 — needs the `server`/`test` Host groups to target

**Status:** ready-for-agent

- [ ] `deploy_docker.py` installs and enables the Docker engine on `server`
- [ ] The Deploy file can be run in isolation against `server` without touching `pi`
- [ ] `--check` reports pending operations without installing anything
- [ ] Re-running against an already-provisioned `server` reports zero pending operations
