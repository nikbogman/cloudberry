# 08 — Deploy entrypoint

**What to build:** `deploy.py` — the single entrypoint composing all Concerns (Tailscale, Docker, Caddy, Control Pi API, Control server API) per Host group. Running it converges both `pi` and `server` to their declared state in one command; a single Concern can also be targeted against a single Host group without re-running everything else. Supports `--check` for a full or targeted dry run. Only ever runs when explicitly invoked (ADR-0009) — no CI, cron, or other automatic trigger exists anywhere in the repo.

**Blocked by:** 02, 03, 05, 06, 07 — composes every Concern that must exist first

**Status:** ready-for-agent

- [ ] `python deploy.py` converges both `pi` and `server` to their declared state by composing the relevant Concerns per Host group in one command
- [ ] A single Concern can be targeted against a single Host group (e.g. just Caddy, just on `pi`) without invoking the rest of the Deploy
- [ ] `--check` previews pending operations — for the full Deploy or a targeted Concern — without touching any device
- [ ] Nothing in the repo (no CI config, no cron, no on-push hook) invokes `deploy.py` automatically — it only runs when a human runs it from the dev machine
