# 08 — Deploy entrypoint

**What to build:** `deploy.py` — the single entrypoint composing all Deploy files (Tailscale, Docker, Caddy, Control Pi API, Control server API) per Host group. Running it converges both `pi` and `server` to their declared state in one command; a single Deploy file can also be targeted against a single Host group without re-running everything else. Supports `--check` for a full or targeted dry run. Only ever runs when explicitly invoked (ADR-0009) — no CI, cron, or other automatic trigger exists anywhere in the repo.

**Blocked by:** 02, 03, 05, 06, 07 — composes every Deploy file that must exist first

**Status:** ready-for-human

- [x] `python deploy.py` converges both `pi` and `server` to their declared state by composing the relevant Deploy files per Host group in one command
- [x] A single Deploy file can be targeted against a single Host group (e.g. just Caddy, just on `pi`) without invoking the rest of the Deploy
- [x] `--check` previews pending operations — for the full Deploy or a targeted Deploy file — without touching any device
- [x] Nothing in the repo (no CI config, no cron, no on-push hook) invokes `deploy.py` automatically — it only runs when a human runs it from the dev machine

## Comments

Implemented in `pyinfra/deploy.py` using `pyinfra.local.include(...)` (the
alternate, file-inclusion composition mechanism `docs/agents/pyinfra.md`
describes) to compose all five Deploy files in dependency order:
tailscale, docker, both Control APIs, then Caddy. Per-Deploy-file
targeting isn't special-cased here — it's an inherent property of how
each Deploy file gates itself on `common.has_device_kind`, so `pyinfra
inventory.py deploy_caddy.py --limit pi` already works standalone. Nothing
in the repo (no CI workflow files exist at all) invokes `deploy.py`.

Verified via `uv run pyinfra <local-inventory> deploy.py --dry` against
`@local`, for device_kind=pi (tailscale + control-pi-api + caddy operations
appear, docker/control-server-api correctly absent) and device_kind=server
(tailscale + docker + control-server-api appear, the pi-only pieces
correctly absent) — each operation clearly labeled with its source file
by `local.include`. Not run against real `pi`/`server` together — no
physical hardware in the sandbox this was built in.
