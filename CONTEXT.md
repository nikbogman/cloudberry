# Homelab Control System

Domain glossary for the control plane that lets the user wake, monitor, and suspend the main compute host from anywhere on the tailnet.

## Language

### Control plane

**Gateway**:
The always-on device (currently a Pi Zero) that fronts the whole system: hosts the UI, sends Wake-on-LAN packets, and reverse-proxies all workload traffic to the Compute host — all three the Gateway API's job (ADR-0016). Named for its role — the tailnet's single entry point — not the hardware it happens to run on.
_Avoid_: Pi, Pi Zero (as a role name — fine as a literal hardware reference)

**Compute**:
The main machine that sleeps/wakes and runs the actual workloads (Immich, AI agents, etc.). Named for its role in the control plane — the thing that does compute work and gets suspended/woken — not the hardware it happens to be.
_Avoid_: server, main server, worker (implies it works for the Gateway, which is backwards — the Gateway serves it by waking it)

**UI**:
The browser-facing app on the Gateway that shows Compute reachability and offers Wake/Suspend actions. The only user-facing surface in the system.
_Avoid_: Control UI, Panel, dashboard, frontend

**Gateway API**:
The backend on the Gateway, serving three jobs from one process (ADR-0016, superseding a prior Caddy/Gateway-API split): it serves the UI's static files, sends an explicit Wake-on-LAN packet when the UI's Wake button is pressed, and reverse-proxies one fixed path (`/server*`) to a single upstream on the Compute host — not one route per workload service (Immich, AI agents, etc.); which path reaches which service is entirely the Compute host's own reverse proxy's concern, invisible to the Gateway (ADR-0011). That proxy route also triggers the wake action itself on any request while Compute is asleep, and holds the request until it responds (ADR-0012) — a second, automatic trigger for the same physical wake action, distinct from the deliberate Wake button. Same-origin with the UI; not reachable from off-tailnet.
_Avoid_: Pi API, Control Gateway API, wake service, Gateway proxy, Pi proxy, Auto-wake proxy, WoL plugin, wake proxy

**Compute API**:
The backend on the Compute host that exposes the Suspend action and a reachability health check. A distinct origin from the UI, fronted by its own `tailscale serve` instance.
_Avoid_: Server API, Control Compute API, suspend service

**Reachable**:
Whether the Compute host currently responds to the Compute API's health check. The UI polls this and uses it to decide which of Wake/Suspend is actionable.
_Avoid_: online, up, awake

**Wake**:
Sending a Wake-on-LAN magic packet to bring Compute out of suspend. Always sent by the Gateway API, triggered two ways: deliberately, via the UI's Wake button, or automatically, via the Gateway API's own `/server*` proxy route triggering the same wake action on any proxied request while asleep (ADR-0012).

**Suspend**:
Putting Compute into suspend-to-RAM. The only sleep state this system supports — full shutdown (ACPI S5) is explicitly out of scope, since WoL after full power-off is unreliable across BIOS/NIC configurations.
_Avoid_: shutdown, sleep, power off

**Identity header**:
The `Tailscale-User-Login` header injected by `tailscale serve` in front of both the Gateway and Compute. The entire authentication mechanism for both apps — tailnet membership is also the entire authorization boundary, with no separate allow-list. One narrow exception: the Gateway API's wake action also accepts a caller on `127.0.0.1` in place of this header, for its own proxy route's same-device trigger (ADR-0004, ADR-0012).
_Avoid_: auth token, login header

### Provisioning

**Deploy**:
A single run of pyinfra against the inventory that converges the Gateway and Compute to their declared state. Run manually, on demand, from the dev machine — never automatic or scheduled.
_Avoid_: playbook run, apply

**Deploy file**:
A single-responsibility pyinfra file scoped to one piece of infrastructure or one app (e.g. Tailscale, Docker, the Gateway API). The unit a Deploy can be targeted or dry-run against in isolation. A piece reused across Deploy files (e.g. the git-pull-plus-systemd-unit pattern shared by both the Gateway API and Compute API) is written as an `@deploy`-decorated deploy function and imported where needed.
_Avoid_: Concern, role, task, module

**Host group**:
A pyinfra inventory grouping of devices by responsibility — `gateway` and `compute`. Determines which Deploy files apply to which device.
_Avoid_: role (in the Ansible sense), pi, server (old host group names)
