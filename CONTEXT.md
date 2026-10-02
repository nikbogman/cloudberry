# Cloudberry

Domain glossary for this monorepo. One service so far — the Platform, which lets the user wake, monitor, and suspend the Sleeper from anywhere — plus the root [`Taskfile.yml`](Taskfile.yml) that converges every device.

## Language

### System

**Platform**:
The Go module at the repo root (`cmd/`, `internal/`, `ui/`): Edge, the Waker API, the Sleeper API and the UI. Not the Deploy tooling (`Taskfile.yml`) or the Stacks.
_Avoid_: waker-service (the old name)

**Edge**:
The public entry point, hosted on Railway: serves the UI and forwards to the Waker API, the Sleeper API and Caddy-fronted Stacks over the tailnet (as its own `tsnet` node). The browser's only origin. No authentication for now — deliberate and temporary.
_Avoid_: proxy, gateway, railway service

**Waker**:
Always-on device (currently a Pi Zero) that sends Wake-on-LAN — the Waker API's only job. On the LAN because the magic packet is a broadcast. Not in the workload traffic path: it's on Wi-Fi, so the hop isn't worth it. Named for its role, not its hardware.
_Avoid_: Gateway (its name until 2026-09-12, when it stopped carrying workload traffic), Pi, Pi Zero, raspberry (its hardware name — see [README.md](README.md#name) — used as a role name)

**Sleeper**:
The machine that sleeps/wakes and runs the actual workloads. Named for its role, not its hardware — and to pair with the Waker.
_Avoid_: Compute (its name until 2026-09-12), server, main server, compute host, worker (implies it works for the Waker — backwards), blackberry (its hardware name — see [README.md](README.md#name) — used as a role name)

**UI**:
The browser app served by Edge showing Sleeper reachability and Wake/Suspend actions. The only user-facing surface.
_Avoid_: Control UI, Panel, dashboard, frontend

**Waker API**:
The backend on the Waker: sends WoL on `POST /wake`, nothing else. Carries no workload traffic. Not reachable off-tailnet; Edge forwards to it.
_Avoid_: Gateway API, Pi API, wake service, WoL plugin

**Sleeper API**:
The backend on the Sleeper exposing Suspend and a reachability health check. Carries no workload traffic and knows nothing about Docker. Fronted by its own `tailscale serve` instance; only Edge calls it.
_Avoid_: Compute API (its name until 2026-09-12), Server API, suspend service

**Reachable**:
Whether Sleeper currently responds to the Sleeper API's health check. The UI polls this to decide which of Wake/Suspend is actionable.
_Avoid_: online, up, awake

**Wake**:
Sending a WoL magic packet to bring Sleeper out of suspend. Always sent by the Waker API, and only deliberately — via the UI's Wake button. Workload traffic no longer passes through the Waker, so nothing wakes Sleeper automatically.

**Suspend**:
Putting Sleeper into suspend-to-RAM — the only sleep state supported. Full shutdown (ACPI S5) is out of scope: WoL after full power-off is unreliable across BIOS/NIC configs. Triggered manually via the UI's Suspend button.
_Avoid_: shutdown, sleep, power off

**Identity header**:
`Tailscale-User-Login`, injected by `tailscale serve` in front of both apps. The entire auth mechanism behind Edge — tailnet membership is the authorization boundary, no separate allow-list. Edge's calls carry the login of the user who owns Edge's tailnet node. Exception: the Waker API's wake action also accepts a `127.0.0.1` caller in its place — originally for the in-process auto-wake trigger on the device's old proxy route; with that route gone it now only covers a shell on the Pi itself.
_Avoid_: auth token, login header

### Provisioning

**Deploy**:
A `task` run (repo root `Taskfile.yml`) that converges the Waker and Sleeper over ssh. Manual only — never automatic or scheduled.
_Avoid_: playbook run, apply


### Workloads

**Stack**:
A user-facing workload (an AI agent, etc.) at `stacks/<name>/`, deployed as plain Docker Compose directly to blackberry — outside this repo's Deploy, unknown to the Waker/Sleeper services. See [`docs/stacks.md`](docs/stacks.md).
_Avoid_: service (this repo's term for the Waker/Sleeper API processes), workload (fine informally, but "stack" is the file/directory unit)

**blackberry**:
Sleeper's hardware name, used deliberately inside Stacks docs and config instead of "Sleeper" — Stacks is outside the Sleeper role's domain. A documented exception to the _Avoid_ note on **Sleeper** above, not an oversight.
