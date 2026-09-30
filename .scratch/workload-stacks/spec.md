# Spec: Workload Stacks

Status: ready-for-agent

## Problem Statement

The user wants to run user-facing workloads (Jellyfin, an AI agent, and others later) on blackberry, reachable over the tailnet, without pulling them into this repo's pyinfra-managed infrastructure or coupling them to the Waker/Sleeper services. Today there's no place for these workloads to live, no agreed way to deploy or inspect them, and no routing story now that the old Docker-aware reverse proxy (`homelab.route` labels, idle watcher, Caddy auto-wake-proxy) has been removed.

## Solution

Introduce `stacks/<name>/` as plain Docker Compose projects, deployed and inspected from a dev machine through one SSH-based Docker context (`blackberry`) — never through pyinfra, never through a repo script. Each stack keeps its own secrets (`.env`, box-only, untracked) and persistent data (`/srv/stacks/<name>/`, created manually). A Caddy stack fronts the others using the `caddy-tailscale` plugin so each workload gets its own real tsnet hostname (e.g. `jellyfin.<tailnet>.ts.net`) instead of a shared-hostname path prefix. This keeps pyinfra, the Waker API, and the Sleeper API exactly as Docker-ignorant as they are today.

This is a convention + first-implementation spec: the ADR, the agent-facing doc, and the glossary entry already exist (`docs/adr/DECISIONS.md`, `docs/agents/stacks.md`, `CONTEXT.md`). What's left is writing the first real stacks against that convention and doing the one-time environment setup they depend on.

## User Stories

1. As the user, I want a `stacks/<name>/` convention documented once, so that adding a new workload later doesn't require re-deriving folder layout, secrets handling, or routing from scratch.
2. As the user, I want to deploy any stack with a single `docker compose --context blackberry` command, so that I don't need pyinfra or a bespoke deploy script for workloads that aren't part of this repo's own infrastructure.
3. As the user, I want to browse what's running on blackberry from Docker Desktop or the VS Code Docker extension, so that I have a visual view of workload containers without SSHing in.
4. As the user, I want each workload's persistent data under a fixed path (`/srv/stacks/<name>/`), so that backups and disk inspection have one predictable location per stack.
5. As the user, I want each workload's secrets to live in a per-stack `.env` created directly on blackberry, so that credentials are never committed to git and never pass through any repo tooling.
6. As the user, I want stacks to resume automatically after blackberry reboots or wakes from suspend, so that I don't have to manually restart containers after every wake.
7. As the user, I want each workload reachable at its own tsnet hostname (e.g. `jellyfin.<tailnet>.ts.net`), so that apps that refuse or mishandle path prefixes (Immich) still work, and apps that tolerate prefixes (Jellyfin) aren't forced into one anyway.
8. As the user, I want a working Jellyfin stack as the first proof of the convention, so that I can validate the whole path (compose file, secrets, persistence, restart, routing) end to end on the simplest workload before tackling anything harder.
9. As the user, I want a working Caddy stack with the `caddy-tailscale` plugin configured and verified against real docs, so that the per-workload-hostname routing story is proven, not just designed.
10. As the user, I want the `blackberry` Docker context and the shared external Docker network documented as one-time manual setup steps, so that setting up a new dev machine or recovering from a wiped one is a known, repeatable procedure rather than tribal knowledge.
11. As the user, I want the AI agent workload's requirements (GPU passthrough, model storage path, which agent) scoped before it's built, so that its compose file isn't written against guesses.
12. As the user, I want Immich to stay explicitly out of scope, so that nobody accidentally starts building it under this convention before it's actually decided.
13. As the user, I want the Waker/Sleeper services' existing "knows nothing about Docker" boundary preserved, so that this feature doesn't quietly reintroduce the coupling that was just removed (`homelab.route` labels, idle watcher, auto-wake-proxy).
14. As the user, I want the glossary to keep "blackberry" as the correct term inside Stacks docs (rather than "Sleeper"), so that agents reading Stacks docs don't misapply the Sleeper glossary's "avoid blackberry" rule from a different domain.

## Implementation Decisions

Already decided and recorded (see `docs/adr/DECISIONS.md`, `docs/agents/stacks.md`, `CONTEXT.md` — do not re-litigate):

- **Scope**: `stacks/<name>/docker-compose.yml`, plain compose only, no new repo service, no application code added to this repo. In scope for this pass: Caddy (ingress) and Jellyfin. The AI agent workload is named but not yet scoped (see User Story 11 / Out of Scope). Immich and the repo's own Waker/Sleeper/UI services are explicitly excluded.
- **Deploy + visibility**: one SSH-based Docker context, `blackberry` (`docker context create blackberry --docker "host=ssh://user@blackberry"`). Deploy: `docker compose --context blackberry -f stacks/<name>/docker-compose.yml up -d`. Same context is used by Docker Desktop / VS Code Docker extension for read-only visibility. No pyinfra involvement, no new deploy script — this is a manual, one-time context creation on each dev machine, not something the repo automates.
- **Secrets**: per-stack `.env` in `stacks/<name>/`, created directly on blackberry over SSH, box-only, never committed, never rendered by any repo tooling (no SOPS/age — considered and rejected as overkill).
- **Persistence**: fixed convention `/srv/stacks/<name>/` on blackberry, created manually per stack, not pyinfra-managed.
- **Restart behavior**: `restart: unless-stopped` per service — native Docker Compose, no extra tooling needed since dockerd is already pyinfra-enabled at boot.
- **Routing**: Caddy stack runs the `caddy-tailscale` plugin so each site binds its own tsnet hostname (`bind tailscale/<name>` in the Caddyfile, `ephemeral: false`, `state_dir` under `/srv/stacks/caddy/`). Each site's Tailscale auth key is a secret in Caddy's own `.env`. This was verified against the plugin's docs (context7, `/tailscale/caddy-tailscale`) but not yet tested live — first real deploy is where that gets proven.
- **Shared network**: all Caddy-fronted stacks join one shared external Docker network, created manually once on blackberry. Exact network name is an implementation detail left to whoever writes the first compose files (Jellyfin + Caddy).
- **Naming boundary**: Stacks-related docs/config use **blackberry** (the hardware name), never **Sleeper** — Sleeper is the waker-service's role name and is out of the Stacks domain by design. This is a deliberate, documented exception to the Sleeper glossary entry in `CONTEXT.md`, not an oversight.
- **First implementation order**: Jellyfin first (simplest, no GPU/model complexity), Caddy alongside or immediately after it (needed to prove routing). The AI agent stack comes after Jellyfin+Caddy prove the pattern, once its own requirements are scoped.
- **One-time environment setup** (not repo files, done by hand): create the `blackberry` Docker context on the dev machine; create `/srv/stacks/` and the shared Docker network on blackberry; provision the first `caddy-tailscale` auth key(s).

## Testing Decisions

- No automated test seam. `stacks/` is plain Docker Compose config that this repo's own tooling (pyinfra, CI) deliberately never touches or knows exists — per the ADR's "nothing in this repo talks to the Docker daemon or knows a stack exists" boundary. Adding a lint/CI script would itself be new tooling that knows stacks exist, which is the thing this design avoids.
- Verification per stack is manual: `docker compose -f stacks/<name>/docker-compose.yml config` to catch syntax errors before deploying, then `docker compose --context blackberry -f stacks/<name>/docker-compose.yml up -d` and a live check that the workload answers at its tsnet hostname.
- No prior art in this repo for compose-file tests — this is the first Docker Compose content added.

## Out of Scope

- Immich (explicitly rejected for this pass — see ADR).
- Any change to the Waker API, Sleeper API, or UI — they remain Docker-ignorant.
- Reintroducing any form of Docker-aware routing in the Waker/Sleeper services (`homelab.route` labels, idle watcher, auto-wake-proxy) — that was deliberately removed on this branch.
- SOPS/age or any secrets-at-rest encryption scheme for `.env` files.
- pyinfra involvement of any kind — no Deploy file, no fact-checking, no dry-run tier for stacks.
- Scoping or building the AI agent workload itself (which agent, GPU passthrough, model storage) — tracked as a follow-up, not part of this spec.
- CI/lint automation for compose files (see Testing Decisions).
- The CSRF exposure on `/wake` and `/suspend` (tracked separately in memory as `homelab-csrf-wake-exposure`) — unrelated, pre-existing, not touched by this feature.

## Further Notes

- Full background: `/tmp/homelab-docker-stacks-handoff.md` (this session's handoff), and long-term memory notes `workload-stacks-adopted.md`, `workload-stacks-dropped.md` (an earlier, broader routing redesign that was designed then reverted — two of its findings were reused here: per-app path-prefix support is inconsistent across Jellyfin/Immich, and suspend is a sharper constraint than routing for always-on-needing workloads like Home Assistant or file sync).
- `docs/agents/stacks.md` is the short-form agent-facing reference for this convention going forward; read it before touching anything under `stacks/`.
- If new terminology comes up while writing the actual compose files, extend `CONTEXT.md`'s Workloads section rather than introducing ad hoc terms; if a new architectural decision is needed, append a `##` section to `docs/adr/DECISIONS.md` (repo convention: one running file, not one file per decision).
