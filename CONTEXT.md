# Homelab Control System

Domain glossary for the control plane that lets the user wake, monitor, and suspend the main compute host from anywhere on the tailnet.

## Language

### Control plane

**Gateway**:
Always-on device (currently a Pi Zero) fronting the system: hosts the UI and sends Wake-on-LAN — both the Gateway API's job. Not in the workload traffic path: it's on Wi-Fi, so the hop isn't worth it. Named for its role, not its hardware.
_Avoid_: Pi, Pi Zero (as a role name)

**Compute**:
The machine that sleeps/wakes and runs the actual workloads. Named for its role, not its hardware.
_Avoid_: server, main server, worker (implies it works for the Gateway — backwards)

**UI**:
The browser app on the Gateway showing Compute reachability and Wake/Suspend actions. The only user-facing surface.
_Avoid_: Control UI, Panel, dashboard, frontend

**Gateway API**:
The backend on the Gateway: serves the UI's static files and sends WoL. Carries no workload traffic — browsers reach the Compute API's own reverse proxy directly, so Wake has exactly one trigger, the UI's Wake button. Same-origin with the UI; not reachable off-tailnet.
_Avoid_: Pi API, wake service, Gateway proxy, WoL plugin

**Compute API**:
The backend on Compute exposing Suspend and a reachability health check — and Compute's own reverse proxy for workload traffic, routing to Docker containers by label. A distinct origin from the UI, fronted by its own `tailscale serve` instance.
_Avoid_: Server API, suspend service

**Reachable**:
Whether Compute currently responds to the Compute API's health check. The UI polls this to decide which of Wake/Suspend is actionable.
_Avoid_: online, up, awake

**Wake**:
Sending a WoL magic packet to bring Compute out of suspend. Always sent by the Gateway API, and only deliberately — via the UI's Wake button. Workload traffic no longer passes through the Gateway, so nothing wakes Compute automatically.

**Suspend**:
Putting Compute into suspend-to-RAM — the only sleep state supported. Full shutdown (ACPI S5) is out of scope: WoL after full power-off is unreliable across BIOS/NIC configs. Triggered manually via the UI's Suspend button.
_Avoid_: shutdown, sleep, power off

**Identity header**:
`Tailscale-User-Login`, injected by `tailscale serve` in front of both apps. The entire auth mechanism — tailnet membership is the entire authorization boundary, no separate allow-list. Exception: the Gateway API's wake action also accepts a `127.0.0.1` caller in its place — originally for the Gateway's own proxy route; with that route gone it now only covers a shell on the Pi itself.
_Avoid_: auth token, login header

### Provisioning

**Deploy**:
A single pyinfra run against the inventory that converges the Gateway and Compute to their declared state. Manual only — never automatic or scheduled.
_Avoid_: playbook run, apply

**Deploy file**:
A single-responsibility pyinfra file scoped to one piece of infrastructure or app (e.g. Tailscale, the Gateway API). The unit a Deploy can be targeted or dry-run against in isolation. Logic shared across Deploy files (e.g. `go_binary_systemd_service`, used by both the Gateway API and Compute API) lives in a plain importable module like `go_build.py`.
_Avoid_: Concern, role, task, module

**Host group**:
A pyinfra inventory grouping of devices by responsibility — `gateway` and `compute`. Determines which Deploy files apply to which device.
_Avoid_: role (Ansible sense), pi, server (old group names)
