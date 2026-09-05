# Homelab Control System

Domain glossary for the control plane that lets the user wake, monitor, and suspend the main compute host from anywhere on the tailnet.

## Language

### Control plane

**Gateway**:
Always-on device (currently a Pi Zero) fronting the system: hosts the UI, sends Wake-on-LAN, reverse-proxies workload traffic to Compute — all three the Gateway API's job. Named for its role, not its hardware.
_Avoid_: Pi, Pi Zero (as a role name)

**Compute**:
The machine that sleeps/wakes and runs the actual workloads. Named for its role, not its hardware.
_Avoid_: server, main server, worker (implies it works for the Gateway — backwards)

**UI**:
The browser app on the Gateway showing Compute reachability and Wake/Suspend actions. The only user-facing surface.
_Avoid_: Control UI, Panel, dashboard, frontend

**Gateway API**:
The backend on the Gateway: serves the UI's static files, sends WoL, and reverse-proxies `/server*` to a single upstream on Compute — not one route per workload service; which path reaches which service is the Compute API's concern. That proxy route also wakes Compute on any request while it's asleep — a second trigger, distinct from the Wake button. Same-origin with the UI; not reachable off-tailnet.
_Avoid_: Pi API, wake service, Gateway proxy, WoL plugin

**Compute API**:
The backend on Compute exposing Suspend, Hold, and a reachability health check — and Compute's own reverse proxy for workload traffic, routing to Docker containers by label. Also runs the idle watcher that triggers automatic Suspend. A distinct origin from the UI, fronted by its own `tailscale serve` instance.
_Avoid_: Server API, suspend service

**Reachable**:
Whether Compute currently responds to the Compute API's health check. The UI polls this to decide which of Wake/Suspend is actionable.
_Avoid_: online, up, awake

**Wake**:
Sending a WoL magic packet to bring Compute out of suspend. Always sent by the Gateway API — deliberately via the UI's Wake button, or automatically via the `/server*` proxy route on any request while asleep.

**Suspend**:
Putting Compute into suspend-to-RAM — the only sleep state supported. Full shutdown (ACPI S5) is out of scope: WoL after full power-off is unreliable across BIOS/NIC configs. Triggered manually via the UI's Suspend button, or automatically by the Compute API's idle watcher when no workload container activity, no proxied HTTP traffic, and no active Hold have been observed for the idle timeout.
_Avoid_: shutdown, sleep, power off

**Hold**:
A time-limited lock on the Compute API that blocks automatic Suspend regardless of idle signals, acquired via `POST /hold` (fixed 30-minute TTL, renewed by calling again) and released early via `DELETE /hold`. Exists so a long-running operation like a backup or restore can't be suspended out from under itself.
_Avoid_: lock, pause, pin

**Identity header**:
`Tailscale-User-Login`, injected by `tailscale serve` in front of both apps. The entire auth mechanism — tailnet membership is the entire authorization boundary, no separate allow-list. Exception: the Gateway API's wake action and the Compute API's Hold action also accept a `127.0.0.1` caller in its place — for the proxy route's same-device trigger, and for backup/restore scripts running locally on Compute with no browser session to carry an identity.
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
