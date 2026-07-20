# 02 — Tailscale Concern

**What to build:** `deploy_tailscale.py` — a single-responsibility Concern that joins a target device to the tailnet, reading the auth key from the `TAILSCALE_AUTH_KEY` environment variable on the dev machine at Deploy time (ADR-0010). Targetable and dry-runnable in isolation against either `pi` or `server`.

**Blocked by:** 01 — needs the `pi`/`server`/`test` Host groups to target

**Status:** ready-for-agent

- [ ] `deploy_tailscale.py` installs Tailscale and joins the tailnet on a target host, using an auth key read from `TAILSCALE_AUTH_KEY` — the key is never written to a file in the repo
- [ ] The Concern can be run in isolation against just `pi` or just `server`, without touching the other
- [ ] `--check` reports the operations that would run without joining the tailnet or mutating the device
- [ ] Re-running against a host that's already joined reports zero pending operations
