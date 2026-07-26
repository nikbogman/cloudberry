# 04 — Suspend action (Control server API + UI Suspend button)

**What to build:** The Control UI's Suspend button calls the Control server API, which puts the main server into suspend-to-RAM (the only supported sleep state — no full shutdown, per ADR-0002). The button is disabled while the server is already Unreachable, using the reachability state from ticket 02. Suspend requested/succeeded/failed events are logged with the caller's identity via the Grafana Alloy client from ticket 01. Demoable end-to-end: with the server awake, press Suspend in the UI and see it suspend and the event logged.

**Blocked by:** 02 — Reachability status (health check + UI display)

**Status:** ready-for-human

- [x] Control server API exposes an endpoint that triggers suspend-to-RAM, protected by the identity-header check from ticket 01
- [x] Tests exercise the HTTP boundary via Flask's test client, mocking only the suspend system call/subprocess
- [x] Control UI's Suspend button is disabled whenever the latest known state is Unreachable, and enabled otherwise
- [x] Pressing Suspend calls the Control server API and does not implement its own no-op guard beyond what the UI's enable/disable state already provides — the API always attempts the action and reports the outcome
- [x] Suspend requested / succeeded / failed events are logged via the Grafana Alloy client from ticket 01, tagged with the caller's identity from the header
- [x] No full-shutdown (ACPI S5) code path exists anywhere in this endpoint

## Comments

Implemented in `services/control_server_api` and `services/control_ui`, commit `982cdb0`.
