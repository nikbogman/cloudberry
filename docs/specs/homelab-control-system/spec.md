Status: ready-for-agent

# Homelab Control System

## Problem Statement

The user's workloads (Immich, AI agents, and other Docker Compose stacks) run on a main server that isn't always on — it needs to sleep to save power when idle. Today there's no way to remotely wake the server, no visibility into whether it's currently reachable, and no way to remotely suspend it — all while relying on Tailscale as the only network path in when away from the local network. Every interaction currently requires physically being near the server.

## Solution

A small control plane split across the two always-reachable-via-tailnet devices:

- **Control UI**, served from the Pi Zero (which is always on), shows whether the main server is Reachable and offers Wake and Suspend actions.
- **Control Pi API**, on the Pi Zero, sends an explicit Wake-on-LAN packet when the user presses Wake.
- **Control server API**, on the main server, exposes a Suspend action and a health check the Control UI polls for Reachable status.
- **Auto-wake proxy** (Caddy's WoL plugin on the Pi Zero) reverse-proxies every Docker service on the main server and transparently wakes it on any incoming request while asleep — independent of the Control UI/API, so ordinary access to Immich etc. doesn't require visiting the control panel first.

Tailscale's injected `Tailscale-User-Login` identity header is the entire authentication mechanism for both Control APIs — no login screen, password, or token store anywhere in the system. See `CONTEXT.md` for the full glossary and `docs/adr/` for the architectural decisions referenced below.

## User Stories

1. As a tailnet user, I want to see at a glance whether the main server is Reachable, so that I know whether I need to wake it before using its services.
2. As a tailnet user, I want the Control UI's reachability status to update automatically without me refreshing the page, so that I don't act on stale information.
3. As a tailnet user, I want to press a Wake button to send a Wake-on-LAN packet to the main server, so that I can bring it up remotely before I need it.
4. As a tailnet user, I want the Wake button disabled while the server is already Reachable, so that I don't send pointless wake requests.
5. As a tailnet user, I want to press a Suspend button to put the main server into suspend-to-RAM, so that I can save power when I'm done using it.
6. As a tailnet user, I want the Suspend button disabled while the server is already unreachable, so that I don't act on a server that's already asleep.
7. As a tailnet user, I want to open any hosted service on the main server (e.g. Immich) directly, without visiting the Control UI first, and have it wake automatically if asleep, so that the control panel isn't a mandatory extra step for ordinary use.
8. As a tailnet user, I want that first request to a sleeping service to eventually succeed once the server wakes, rather than fail outright, so that I don't have to guess when to retry.
9. As a tailnet user, I want every wake and suspend action logged with who triggered it and the outcome, so that I have an audit trail if something unexpected happens to my server.
10. As a tailnet user, I want reachability state changes logged, so that I can see server uptime/availability history in Grafana.
11. As the operator, I want both Control APIs to reject any request that doesn't carry a valid Tailscale identity header, so that the control plane isn't exposed to anyone off my tailnet.
12. As the operator, I want every component's task/role definitions to assert they are not bound to any address reachable off-tailnet, so that a configuration mistake can't silently widen the attack surface — this is a hard requirement per the auth model (see ADR-0004).
13. As the operator, I want the Control server API to reachable-check without needing any credentials beyond tailnet membership, so that any of my own devices can check status without extra setup.
14. As the operator, I want the Control UI and Control Pi API calls to work without a CORS entry (same-origin on the Pi), and the Control server API to require an explicit CORS allow-list entry for the Control UI's origin, so that the direct browser-to-API model (ADR-0001) works without a relay.
15. As the operator, I want Suspend to be the only sleep action anywhere in the system (no full shutdown), so that Wake-on-LAN resume stays fast and reliable (ADR-0002).
16. As the operator, I want it documented that Wake-on-LAN requires the Pi and server to stay on the same broadcast domain, so that I don't unknowingly break wake functionality by moving a device to a different VLAN later (ADR-0003).
17. As the operator, I want both Control APIs to remain fully stateless, so that there's no local database or volume to provision, back up, or lose.
18. As a future operator, I want autosuspend (server suspends itself after 1 hour of inactivity) noted as a planned-but-unbuilt feature, so that it isn't forgotten, without its detailed behavior being designed prematurely.

## Implementation Decisions

**Components**:
- Control UI: static TypeScript SPA (Vite, framework-light — no React/Svelte), built to static files, served by Caddy on the Pi Zero via `tailscale serve`.
- Control Pi API: Flask app on the Pi Zero. Same origin as the Control UI (path-routed under one `tailscale serve` app) — no CORS needed between them. Endpoint(s): trigger Wake (send WoL magic packet to the main server's MAC address).
- Control server API: Flask app on the main server, its own `tailscale serve` instance (distinct origin). Endpoints: trigger Suspend (suspend-to-RAM only, per ADR-0002); health check for Reachable status.
- Auto-wake proxy: Caddy's Wake-on-LAN plugin on the Pi Zero, configured as a reverse proxy in front of every Docker Compose service on the main server. On a request while asleep: sends the WoL packet and holds/retries the proxied request until the server responds (transparent, not an error) — see ADR-0005 for why this is independent of the Control Pi API's wake path.

**Auth model**: `tailscale serve` in front of both the Pi and the server injects `Tailscale-User-Login`. This is the entire authentication AND authorization boundary — any tailnet-authenticated identity may act, no allow-list (ADR-0004). Every role/task definition for every component must include an explicit assertion that the app is not bound to any address reachable off-tailnet (loopback or tailnet interface only) — a hard requirement, not a documentation note.

**CORS**: only the Control server API needs a CORS allow-list entry, permitting the Control UI's origin (ADR-0001, and the same-origin Pi topology above).

**Reachability**: Control UI polls the Control server API's health check on an interval (~10–15s) while the page is open. Wake/Suspend buttons are enabled/disabled based on the latest known Reachable state. Neither Control API implements its own redundancy guard against repeated/no-op actions — they always attempt the requested action and report the outcome; the UI is the only layer that prevents the nonsensical case.

**Persistence**: none. Both Control APIs are fully stateless — no database, no local history store. All history/audit trail lives in Grafana Alloy.

**Logging**: all events shipped to Grafana Alloy (free tier). Concrete event list: wake requested / succeeded / failed; suspend requested / succeeded / failed; reachability changed; errors. Each wake/suspend event is tagged with the caller's identity from the Tailscale identity header.

**Provisioning**: both devices are configured via pyinfra. Playbook/role structure is not detailed in this spec (see Out of Scope).

## Testing Decisions

Tests should exercise external behavior at the highest available seam, not internal implementation details:

- **Control Pi API / Control server API**: test at the HTTP boundary using Flask's test client — real routes, real request/response handling, real identity-header auth logic. Mock only the true external side effects: the raw WoL socket send (Control Pi API) and the suspend system call/subprocess (Control server API). This is the one seam per app: the OS/network boundary.
- **Control UI**: test at the component/rendering boundary (button enabled/disabled state, polling-driven status updates), mocking only the `fetch` calls to the two Control APIs. This is the one seam: the network boundary. Do not mock internal state-management functions separately from that boundary.
- No prior art exists in this repo yet (greenfield) — these seams are the first testing convention established for the project and should be treated as precedent for any future component.

## Out of Scope

- Autosuspend implementation (the "suspend after 1 hour of inactivity" behavior) — noted as a planned feature (User Story 18) but its trigger definition, "activity" definition, and implementation are explicitly deferred to a future spec.
- Full shutdown (ACPI S5) support — suspend-to-RAM is the only supported sleep state (ADR-0002).
- Cross-subnet/VLAN Wake-on-LAN (directed broadcast or relay) — both devices must remain on the same broadcast domain (ADR-0003).
- Tailnet identity allow-listing / per-user authorization — tailnet membership alone is the authorization boundary (ADR-0004).
- Local persistence, history stores, or databases on either device.
- pyinfra playbook/role structure and content — provisioning tooling is confirmed but not designed in this spec.
- GitHub Issues migration for this repo's tracker — currently using local markdown under `docs/specs/`; revisit only if `gh` gets installed and configured.

## Further Notes

- Domain vocabulary (Control UI, Control Pi API, Control server API, Auto-wake proxy, Reachable, Wake, Suspend, Identity header) is defined in `CONTEXT.md` at the repo root — use these exact terms, not synonyms, in any implementation issues split out from this spec.
- Architectural decisions referenced throughout (ADR-0001 through ADR-0005) live in `docs/adr/`.
- This spec intentionally covers all three components together rather than as separate specs, since they form one tightly-coupled control plane with a shared auth model, not independent features.
