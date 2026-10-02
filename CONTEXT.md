# Cloudberry

Domain glossary for this monorepo. One service so far — the Platform, which lets the user wake, monitor, and suspend blackberry from anywhere — plus the root [`Taskfile.yml`](Taskfile.yml) that converges every device.

## Language

### Machines

Machines go by their hardware names (see [README.md](README.md#name)) everywhere — docs, config, env vars. Services are named for what they do.

**raspberry**:
Always-on device (a Pi Zero) that runs waker. On the LAN because the magic packet is a broadcast. Not in the workload traffic path: it's on Wi-Fi, so the hop isn't worth it.
_Avoid_: Waker (its role name until 2026-10-02 — now the service's name only), Gateway, Pi

**blackberry**:
The machine that sleeps/wakes and runs hostd and the Stacks.
_Avoid_: Sleeper (its role name until 2026-10-02), Compute, server, main server, worker

### System

**Platform**:
The Go module at the repo root (`cmd/`, `internal/`, `ui/`): edge, waker, hostd and the UI. Not the Deploy tooling (`Taskfile.yml`) or the Stacks.
_Avoid_: waker-service (the old name)

**edge**:
The public entry point, hosted on Railway: serves the UI and forwards to waker, hostd and Caddy-fronted Stacks over the tailnet (as its own `tsnet` node). The browser's only origin. No authentication for now — deliberate and temporary.
_Avoid_: edge-api, proxy, gateway, railway service

**waker**:
The service on raspberry: sends WoL to blackberry on `POST /wake`, nothing else. Carries no workload traffic. Not reachable off-tailnet; edge forwards to it.
_Avoid_: Waker API, waker-api, wake service, WoL plugin

**hostd**:
blackberry's own daemon: exposes Suspend and a reachability health check. Carries no workload traffic and knows nothing about Docker (yet). Fronted by its own `tailscale serve` instance; only edge calls it.
_Avoid_: Sleeper API, sleeper-api, Compute API, suspend service

**UI**:
The browser app served by edge showing blackberry's reachability and Wake/Suspend actions. The only user-facing surface.
_Avoid_: Control UI, Panel, dashboard, frontend

**Reachable**:
Whether blackberry currently responds to hostd's health check. The UI polls this to decide which of Wake/Suspend is actionable.
_Avoid_: online, up, awake

**Wake**:
Sending a WoL magic packet to bring blackberry out of suspend. Always sent by waker, and only deliberately — via the UI's Wake button. Nothing wakes blackberry automatically.

**Suspend**:
Putting blackberry into suspend-to-RAM — the only sleep state supported. Full shutdown (ACPI S5) is out of scope: WoL after full power-off is unreliable across BIOS/NIC configs. Triggered manually via the UI's Suspend button, through hostd.
_Avoid_: shutdown, sleep, power off

**Identity header**:
`Tailscale-User-Login`, injected by `tailscale serve` in front of waker and hostd. The entire auth mechanism behind edge — tailnet membership is the authorization boundary, no separate allow-list. edge's calls carry the login of the user who owns edge's tailnet node. Exception: waker's wake action also accepts a `127.0.0.1` caller in its place — originally for the in-process auto-wake trigger on the device's old proxy route; with that route gone it now only covers a shell on raspberry itself.
_Avoid_: auth token, login header

### Provisioning

**Deploy**:
A `task` run (repo root `Taskfile.yml`) that converges raspberry and blackberry over ssh. Manual only — never automatic or scheduled.
_Avoid_: playbook run, apply

### Workloads

**Stack**:
A user-facing workload (an AI agent, etc.) at `stacks/<name>/`, deployed as plain Docker Compose directly to blackberry — outside this repo's Deploy, unknown to waker and hostd. See [`docs/stacks.md`](docs/stacks.md).
_Avoid_: service (this repo's term for edge/waker/hostd), workload (fine informally, but "stack" is the file/directory unit)
