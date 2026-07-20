# 02 — Reachability status (health check + UI display)

**What to build:** The Control server API exposes a health-check endpoint reporting whether the main server is Reachable. The Control UI polls it every 10-15 seconds while the page is open and displays the live Reachable/Unreachable status without requiring a manual refresh. The health check enforces the identity-header auth from ticket 01 and requires no credentials beyond tailnet membership. Reachability state changes are logged via the Grafana Alloy client from ticket 01, giving uptime/availability history in Grafana. Demoable end-to-end: open the Control UI and watch the status track the server's real up/down state.

**Blocked by:** 01 — Control-plane auth & logging scaffold

**Status:** ready-for-human

- [x] Control server API health-check endpoint reports Reachable status, protected by the identity-header check from ticket 01, tested via Flask's test client
- [x] Control server API asserts it is not bound off-tailnet (using ticket 01's helper)
- [x] Control UI polls the health-check endpoint on a 10-15s interval while the page is open, mocking only the `fetch` call in tests
- [x] Control UI displays current Reachable/Unreachable status and updates automatically as polling results change, with no page refresh needed
- [x] Control server API's CORS allow-list permits the Control UI's origin (per ADR-0001); no CORS entry needed on the Pi side
- [x] Reachability state changes are logged via the Grafana Alloy client from ticket 01

## Comments

Implemented in `services/control_server_api` and `services/control_ui`, commit `35a9cff`.
