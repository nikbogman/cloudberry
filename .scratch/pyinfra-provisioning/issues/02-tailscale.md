# 02 — Tailscale deploy file

**What to build:** `deploy_tailscale.py` — a single-responsibility Deploy file that joins a target device to the tailnet, reading the auth key from the `TAILSCALE_AUTH_KEY` environment variable on the dev machine at Deploy time (ADR-0010). Targetable and dry-runnable in isolation against either `pi` or `server`.

**Blocked by:** 01 — needs the `pi`/`server`/`test` Host groups to target

**Status:** ready-for-human

- [x] `deploy_tailscale.py` installs Tailscale and joins the tailnet on a target host, using an auth key read from `TAILSCALE_AUTH_KEY` — the key is never written to a file in the repo
- [x] The Deploy file can be run in isolation against just `pi` or just `server`, without touching the other
- [x] `--check` reports the operations that would run without joining the tailnet or mutating the device
- [x] Re-running against a host that's already joined reports zero pending operations

## Comments

Implemented in `pyinfra/deploy_tailscale.py`. `--check` is `--dry` in
pyinfra 3.x (`docs/agents/pyinfra.md`'s Gotchas). Idempotent re-run is via
a custom `TailscaleBackendState` fact (`tailscale status --json`) read at
prepare time, gating the `tailscale up` call in plain Python — not
`_if`, since the fact is knowable before any operation in this file
executes. The join step only fires for the real `pi`/`server` groups,
never `test`, so a disposable-container run can't enroll an ephemeral
container into the real tailnet.

Verified via `uv run pyinfra <local-inventory> deploy_tailscale.py --dry`
against the `@local` connector (device_kind=pi and device_kind=server): correct
operation set generated, no exceptions, `TAILSCALE_AUTH_KEY` correctly
not required unless a join is actually pending. Not verified against a
real device's `tailscale up`/re-run idempotency — no physical `pi`/`server`
available in the sandbox this was built in.
