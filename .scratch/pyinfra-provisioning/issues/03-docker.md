# 03 — Docker deploy file

**What to build:** `deploy_docker.py` — a single-responsibility Deploy file that installs and enables the Docker engine on the `server` Host group, so the (out-of-scope) workload Compose stacks have an engine to run on. Targetable and dry-runnable in isolation.

**Blocked by:** 01 — needs the `server`/`test` Host groups to target

**Status:** ready-for-human

- [x] `deploy_docker.py` installs and enables the Docker engine on `server`
- [x] The Deploy file can be run in isolation against `server` without touching `pi`
- [x] `--check` reports pending operations without installing anything
- [x] Re-running against an already-provisioned `server` reports zero pending operations

## Comments

Implemented in `pyinfra/deploy_docker.py`, composing Docker's official
apt-repo instructions (no built-in "install Docker" operation in pyinfra
— `docs/agents/pyinfra.md`'s Gotchas) from `apt.key`/`apt.repo`/
`apt.packages`/`systemd.service`. Gated on `common.has_device_kind("server")` so
it's a no-op for `pi`.

Verified via `uv run pyinfra <local-inventory> deploy_docker.py --dry`
against `@local` for both device_kind=server (full operation set) and device_kind=pi
(confirmed no-op, zero operations). Idempotent re-run against a real
`server` not verified — no physical hardware in the sandbox this was
built in.
