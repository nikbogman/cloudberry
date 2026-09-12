# Homelab

Domain glossary for this monorepo. One service so far — the control plane in [`waker-service/`](waker-service/) that lets the user wake, monitor, and suspend the Sleeper from anywhere on the tailnet — plus the root [`deploy/`](deploy/) that converges every device.

## Language

### Control plane

**Waker**:
Always-on device (currently a Pi Zero) fronting the system: hosts the UI and sends Wake-on-LAN — both the Waker API's job. Not in the workload traffic path: it's on Wi-Fi, so the hop isn't worth it. Named for its role, not its hardware.
_Avoid_: Gateway (its name until 2026-09-12, when it stopped carrying workload traffic), Pi, Pi Zero (as a role name)

**Sleeper**:
The machine that sleeps/wakes and runs the actual workloads. Named for its role, not its hardware — and to pair with the Waker.
_Avoid_: Compute (its name until 2026-09-12), server, main server, compute host, worker (implies it works for the Waker — backwards)

**UI**:
The browser app on the Waker showing Sleeper reachability and Wake/Suspend actions. The only user-facing surface.
_Avoid_: Control UI, Panel, dashboard, frontend

**Waker API**:
The backend on the Waker: serves the UI's static files and sends WoL. Carries no workload traffic — browsers reach workload containers on Sleeper directly. Same-origin with the UI; not reachable off-tailnet.
_Avoid_: Gateway API, Pi API, wake service, WoL plugin

**Sleeper API**:
The backend on the Sleeper exposing Suspend and a reachability health check. Carries no workload traffic and knows nothing about Docker. A distinct origin from the UI, fronted by its own `tailscale serve` instance.
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
`Tailscale-User-Login`, injected by `tailscale serve` in front of both apps. The entire auth mechanism — tailnet membership is the entire authorization boundary, no separate allow-list. Exception: the Waker API's wake action also accepts a `127.0.0.1` caller in its place — originally for the in-process auto-wake trigger on the device's old proxy route; with that route gone it now only covers a shell on the Pi itself.
_Avoid_: auth token, login header

### Provisioning

**Deploy**:
A single pyinfra run against the inventory that converges the Waker and Sleeper to their declared state. Manual only — never automatic or scheduled.
_Avoid_: playbook run, apply

**Deploy file**:
A single-responsibility pyinfra file scoped to one piece of infrastructure or app (e.g. Tailscale, the Waker API). The unit a Deploy can be targeted or dry-run against in isolation. Logic shared across Deploy files (e.g. `go_binary_systemd_service`, used by both the Waker API and Sleeper API) lives in a plain importable module like `go_build.py`.
_Avoid_: Concern, role, task, module

**Host group**:
A pyinfra inventory grouping of devices by responsibility — `waker` and `sleeper`. Determines which Deploy files apply to which device.
_Avoid_: role (Ansible sense), pi, server (old group names)
