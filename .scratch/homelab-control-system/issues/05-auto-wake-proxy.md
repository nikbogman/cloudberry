# 05 — Auto-wake proxy for workload services

**What to build:** Caddy's Wake-on-LAN plugin on the Pi Zero, configured as a reverse proxy in front of every Docker Compose workload service on the main server (Immich, AI agents, etc.). On an incoming request while the server is asleep, it sends the WoL packet and holds/retries the proxied request until the server responds — succeeding transparently rather than failing outright. This operates independently of the Control Pi API's Wake path (ADR-0005), so ordinary access to a workload never requires visiting the Control UI first. Demoable end-to-end: with the server suspended, hit a workload's URL directly and watch it come up and eventually serve the request.

**Blocked by:** None — can start immediately (independent of Control UI/API per ADR-0005)

**Status:** ready-for-agent

- [ ] Caddy's WoL plugin is configured on the Pi Zero, reverse-proxying every current Docker Compose workload on the main server
- [ ] A request to a workload route while the server is asleep triggers a WoL packet automatically, with no interaction with the Control UI or Control Pi API
- [ ] The triggering request is held/retried until the server responds, and then succeeds — it does not fail outright while the server wakes
- [ ] This path is verified to function independently even if the Control UI/Control Pi API are down or unreachable
