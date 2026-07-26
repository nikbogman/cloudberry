Status: ready-for-agent

# Declarative Provisioning with pyinfra

## Problem Statement

Setting up the Pi Zero and the main server today is a manual, undocumented process. There's no single source of truth for what should be installed and configured on each device (Tailscale, Docker, Caddy, the Control APIs), no repeatable way to reproduce a device's configuration from scratch, and every change relies on remembering exactly what was done by hand last time.

## Solution

A declarative provisioning setup using pyinfra, run on demand from the dev machine, that converges both devices to their declared state: Tailscale joined to the tailnet, Docker engine installed on the server, Caddy fully configured (including the Auto-wake proxy and all workload routes), and the Control UI, Control Pi API, and Control server API deployed and running. Tailscale, Docker, Caddy, and each Control API are each their own single-responsibility Deploy file, targetable and dry-runnable in isolation, composed by one Deploy entrypoint. See `CONTEXT.md` (Provisioning section) for vocabulary and `docs/adr/0006` through `0010` for the architectural decisions below.

## User Stories

1. As the operator, I want to run a single command from my dev machine that converges both the Pi and the server to their declared state, so that I never have to remember manual setup steps.
2. As the operator, I want each of Tailscale, Docker, Caddy, Control Pi API, and Control server API to be its own Deploy file, so that a change to one doesn't require reasoning about the others.
3. As the operator, I want to target a single Deploy file against a single Host group (e.g. just Caddy, just on the Pi), so that I can iterate on one piece without re-running the whole Deploy.
4. As the operator, I want to preview what a Deploy would change with `--check` before it touches a real device, so that a bad template or logic error surfaces before it's applied.
5. As the operator, I want to run a Deploy against disposable containers standing in for the `pi` and `server` Host groups, so that I can catch real execution errors (bad package names, invalid templates, wrong command syntax) without risking the physical devices.
6. As the operator, I want re-running a Deploy against a device that's already converged to report zero pending operations, so that idempotency is verifiable, not assumed.
7. As the operator, I want the Tailscale auth key and any other secret read from an environment variable on my dev machine at Deploy time, so that no secret is ever committed to the repo.
8. As the operator, I want the Control Pi API and Control server API code pulled directly from this repo's GitHub remote by each device, checked out to a specific ref, so that what's running is always traceable to a commit.
9. As the operator, I want the Control UI built on my dev machine and only its static output delivered to the Pi Zero, so that the Pi never has to run a Node toolchain.
10. As the operator, I want the Control Pi API and Control server API to run as systemd services enabled on boot, so that they survive a device reboot without manual intervention.
11. As the operator, I want changing a Control API's code or config to trigger a systemd restart of the corresponding service as part of the Deploy, so that a Deploy always leaves the running service in sync with the declared code.
12. As the operator, I want the entire Caddyfile — including workload auto-wake-proxy routes for Immich, AI agents, and future services — templated declaratively by pyinfra, so that there's one source of truth and no config drift on the device.
13. As the operator, I want adding a new workload service's route to require editing the pyinfra repo and redeploying, so that the deployed Caddy config never diverges from what's declared in the repo.
14. As the operator, I want the Deploy to only ever run when I explicitly invoke it, so that nothing about my homelab changes unattended while I'm not watching.
15. As the operator, I want the pyinfra inventory to define `pi` and `server` Host groups (plus a disposable `test` group for container-based dry runs), so that each Deploy file applies only to the devices it's relevant to.
16. As the operator, I want workload Docker Compose stacks (Immich, AI agents) to remain outside pyinfra's scope, so that content changes on the server don't require touching the provisioning repo.
17. As the operator, I want initial device bootstrapping (OS flashing, first boot, enabling SSH, installing my public key) to remain a manual prerequisite outside this feature, so that pyinfra can assume SSH connectivity already works.
18. As the operator, I want it noted (not yet enforced) that the existing hard requirement — no component may bind off-tailnet — is not currently checked by pyinfra, so that this gap isn't forgotten even though it's deferred.

## Implementation Decisions

**Repo layout**: `pyinfra/inventory.py` defines the `pi` and `server` Host groups (plus a `test` group for disposable-container targets). One Deploy file per single-responsibility piece: `deploy_tailscale.py`, `deploy_docker.py`, `deploy_caddy.py`, `deploy_control_pi_api.py`, `deploy_control_server_api.py` — the two Control API Deploy files share a common deploy function for the git-pull-plus-systemd-unit pattern, since the two APIs are different applications on different Host groups despite structural similarity. A single `deploy.py` entrypoint composes the relevant Deploy files per Host group.

**Scope**: pyinfra manages infrastructure and the control plane only — Tailscale, Docker engine, Caddy, Control UI, Control Pi API, Control server API. Workload Docker Compose stacks and OS-level hardening are explicitly out of scope (see Out of Scope).

**Code delivery**: each device (`pi`, `server`) performs a `git clone`/pull of this repo directly, checked out to a specific ref, for the Control Pi API and Control server API (ADR-0006). Exception: the Control UI is built on the dev machine and only the static build output is delivered to the Pi Zero — the Pi never runs npm/Node (ADR-0006).

**Process management**: the Control Pi API and Control server API run as systemd units, installed/enabled/restarted by pyinfra — a deliberate exception to the Docker-based pattern used for workload stacks elsewhere on the homelab (ADR-0007). A Deploy that changes a Control API's code or unit file restarts the corresponding service.

**Caddy config**: pyinfra templates the entire Caddyfile on the Pi, including the Auto-wake proxy's workload routes, as a single declarative source of truth — no hand-edited config file on the device (ADR-0008).

**Secrets**: read from environment variables on the dev machine at Deploy time (starting with `TAILSCALE_AUTH_KEY`) and passed to the relevant operation. Nothing secret is committed to the repo (ADR-0010).

**Trigger model**: manual only. A Deploy runs exclusively when explicitly invoked from the dev machine — no CI pipeline, cron job, or other automatic trigger (ADR-0009).

**Bootstrap**: initial OS flashing, first boot, SSH enablement, and public-key installation are an assumed manual prerequisite for each device, outside this feature's scope.

**Bind-address assertion**: the existing hard requirement (every component asserts it is not bound off-tailnet) is not enforced by pyinfra in this iteration — explicitly deferred, flagged as a follow-up rather than a spec'd behavior (see Further Notes).

## Testing Decisions

Infrastructure-as-code doesn't have application-level unit-test seams — the meaningful checks operate directly through pyinfra's own execution engine, at three tiers, reusing the same mechanism (no mocking):

1. **`--check` (dry-run) pass**: every Deploy file is run in pyinfra's check mode before being applied for real, reporting the operations that would run without mutating anything. Catches template and logic errors early.
2. **Disposable-container stage**: a `test` Host group in the inventory points at throwaway containers standing in for the `pi` and `server` OS bases. Deploys are applied against these first to catch real execution errors (bad package names, invalid templates, wrong command syntax) without risking the physical devices.
3. **Idempotency on the real target**: the actual correctness test. Running a Deploy a second time immediately after the first against the real `pi`/`server` Host groups must report zero pending operations — this is how a Deploy file is verified to be correctly declarative, not merely "ran without error" once.

No prior art exists in this repo for infrastructure testing (greenfield); this three-tier approach establishes the precedent. It follows the same highest-seam philosophy used in the control-system application spec (`docs/specs/homelab-control-system/spec.md`) — test at the highest available boundary, mock only what's truly external — adapted to the fact that here the "external" boundary is the physical device itself, which can't be mocked away without losing the point of the test.

## Out of Scope

- Workload Docker Compose stacks (Immich, AI agents, future stacks) — remain outside pyinfra's management entirely.
- OS-level hardening (users, SSH config, firewall) beyond what's needed to run the in-scope Deploy files.
- Initial device bootstrapping (OS image flashing, first boot, enabling SSH, installing the operator's public key) — assumed manual prerequisite.
- Enforcing the "no off-tailnet bind" hard requirement within pyinfra — stays documentation-only for now; flagged for a future session.
- Any automatic or scheduled Deploy trigger (CI, cron, on-push) — manual only.
- Encrypted-at-rest secrets management (e.g. sops, age, a vault) — env vars are sufficient for a single-operator setup.

## Further Notes

- Domain vocabulary (Deploy, Deploy file, Host group) is defined in the Provisioning section of `CONTEXT.md` — use these exact terms in any implementation issues split out from this spec.
- Architectural decisions referenced throughout (ADR-0006 through ADR-0010) live in `docs/adr/`.
- Revisit ADR-0004-adjacent territory (bind-address enforcement) once this feature is built — right now the hard requirement from the control-system spec has no automated backstop anywhere in the codebase.
- This spec assumes the control-system components it deploys (Control UI, Control Pi API, Control server API) are being built per `docs/specs/homelab-control-system/spec.md`; this feature is about *how they get onto the devices*, not what they do once running.
