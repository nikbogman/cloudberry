# Homelab Control System

Domain glossary for the control plane that lets the user wake, monitor, and suspend the main workload server from anywhere on the tailnet.

## Language

### Control plane

**Control UI**:
The browser-facing app on the Pi Zero that shows server reachability and offers Wake/Suspend actions. The only user-facing surface in the system.
_Avoid_: Panel, dashboard, frontend

**Control Pi API**:
The backend on the Pi Zero that sends an explicit Wake-on-LAN packet when the Control UI's Wake button is pressed. Same-origin with the Control UI; not reachable from off-tailnet.
_Avoid_: Pi API, wake service

**Control server API**:
The backend on the main server that exposes the Suspend action and a reachability health check. A distinct origin from the Control UI, fronted by its own `tailscale serve` instance.
_Avoid_: server API, suspend service

**Auto-wake proxy**:
Caddy's Wake-on-LAN plugin on the Pi Zero, reverse-proxying one fixed path (`/server*`) to a single upstream on the main server — not one route per workload service (Immich, AI agents, etc.); which path reaches which service is entirely the main server's own reverse proxy's concern, invisible to the Pi (ADR-0011). Sends a Wake-on-LAN packet on any request while the server is asleep and holds the request until the server responds. Operates independently of the Control Pi API — a second, automatic trigger for the same physical wake action, distinct from the deliberate Wake button.
_Avoid_: WoL plugin, wake proxy

**Reachable**:
Whether the main server currently responds to the Control server API's health check. The Control UI polls this and uses it to decide which of Wake/Suspend is actionable.
_Avoid_: online, up, awake

**Wake**:
Sending a Wake-on-LAN magic packet to bring the main server out of suspend. Triggered two independent ways: the Control Pi API (deliberate, via the Control UI's Wake button) or the auto-wake proxy (automatic, via any proxied request while asleep).

**Suspend**:
Putting the main server into suspend-to-RAM. The only sleep state this system supports — full shutdown (ACPI S5) is explicitly out of scope, since WoL after full power-off is unreliable across BIOS/NIC configurations.
_Avoid_: shutdown, sleep, power off

**Identity header**:
The `Tailscale-User-Login` header injected by `tailscale serve` in front of both the Pi and the server. The entire authentication mechanism for both apps — tailnet membership is also the entire authorization boundary, with no separate allow-list.
_Avoid_: auth token, login header

### Provisioning

**Deploy**:
A single run of pyinfra against the inventory that converges the Pi and the server to their declared state. Run manually, on demand, from the dev machine — never automatic or scheduled.
_Avoid_: playbook run, apply

**Deploy file**:
A single-responsibility pyinfra file scoped to one piece of infrastructure or one app (e.g. Tailscale, Caddy, the Control Pi API). The unit a Deploy can be targeted or dry-run against in isolation. A piece reused across Deploy files (e.g. the git-pull-plus-systemd-unit pattern shared by both Control APIs) is written as an `@deploy`-decorated deploy function and imported where needed.
_Avoid_: Concern, role, task, module

**Host group**:
A pyinfra inventory grouping of devices by responsibility — `pi` and `server`. Determines which Deploy files apply to which device.
_Avoid_: role (in the Ansible sense)
