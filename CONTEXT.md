# Cloudberry

Domain glossary for this monorepo: the services and UI that let the user wake, monitor, and suspend blackberry from anywhere; the Stacks they wake blackberry for; and the root [`Taskfile.yml`](Taskfile.yml) that converges every device.

## Language

### Machines

Machines go by their hardware names (see [README.md](README.md#name)) everywhere — docs, config, env vars. Services are named for what they do.

**raspberry**:
Always-on device (a Pi Zero) that runs gateway. On the LAN because the magic packet is a broadcast. In the workload path only for gateway's routes; it's on Wi-Fi, so other Stacks skip it.

**blackberry**:
The machine that sleeps/wakes and runs hostd and the Stacks.

### System

**cloudberry**:
Everything in this repo, taken together: both machines, the services, the UI, the Stacks and the Taskfile that deploys to both machines.

**edge**:
The public entry point, hosted on Railway: serves the UI and forwards to gateway and hostd over the tailnet (as its own `tsnet` node). The browser's only origin. No authentication for now — deliberate and temporary.
_Avoid_: edge-api, proxy, railway service. Not "gateway": that's the raspberry service.

**gateway**:
The service on raspberry. Sends Wake on `POST /wake` (the UI's Wake button, forwarded by edge), and fronts the Stacks in its routes file: reverse-proxies each route on its own port, wakes blackberry on request for routes flagged `wake` (holding the request until blackberry is Reachable), and suspends it again after `idle_minutes` without traffic, if gateway did the waking. Routes are `public` (Funnel) or tailnet-only. Before 2026-10-23 this was two services, waker (`/wake`) and gate (the routes); before 2026-09-12 it was also called gateway.
_Avoid_: waker, gate, proxy, reverse proxy, wake service, WoL plugin

**hostd**:
blackberry's own daemon: exposes Suspend and a reachability health check. Carries no workload traffic and knows nothing about Docker (yet). Fronted by its own `tailscale serve` instance; only edge calls it.

**UI**:
The browser app served by edge showing blackberry's reachability and Wake/Suspend actions. The only user-facing surface.

**Reachable**:
Whether blackberry currently responds to hostd's health check. The UI polls this to decide which of Wake/Suspend is actionable.

**Wake**:
Sending a WoL magic packet to bring blackberry out of suspend. Always sent by gateway: on the UI's Wake button, or on a request to a route flagged `wake`. Nothing else wakes blackberry.

**Suspend**:
Putting blackberry into suspend-to-RAM — the only sleep state supported. Full shutdown (ACPI S5) is out of scope: WoL after full power-off is unreliable across BIOS/NIC configs. Triggered via hostd by the UI's Suspend button, or by gateway after its idle timeout (only when gateway auto-woke blackberry).
_Avoid_: shutdown, sleep, power off

**Identity header**:
`Tailscale-User-Login`, injected by `tailscale serve` in front of gateway and hostd. The entire auth mechanism behind edge — tailnet membership is the authorization boundary, no separate allow-list. edge's calls carry the login of the user who owns edge's tailnet node. Exception: gateway's `/wake` also accepts a `127.0.0.1` caller in its place, for a shell on raspberry itself. Auto-wake doesn't use it: it sends WoL in-process.
_Avoid_: auth token, login header

### Provisioning

**Deploy**:
A `task` run (repo root `Taskfile.yml`) that converges raspberry and blackberry over ssh. Manual only — never automatic or scheduled.
_Avoid_: playbook run, apply

### Workloads

**Stack**:
A user-facing workload at `stacks/<name>/`, deployed as plain Docker Compose directly to blackberry — outside this repo's Deploy, unknown to hostd; gateway knows only those in its routes file. See [`stacks/README.md`](stacks/README.md).
_Avoid_: service (this repo's term for edge/gateway/hostd; Platform and waker-service are old names for the set), workload (fine informally, but "stack" is the file/directory unit)
