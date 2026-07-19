# 03 — Wake action (Control Pi API + UI Wake button)

**What to build:** The Control UI's Wake button calls the Control Pi API, which sends an explicit Wake-on-LAN magic packet to the main server's MAC address. The button is disabled while the server is already Reachable, using the reachability state from ticket 02. Wake requested/succeeded/failed events are logged with the caller's identity via the Grafana Alloy client from ticket 01. Demoable end-to-end: with the server asleep, press Wake in the UI and see the WoL packet sent and the event logged.

**Blocked by:** 02 — Reachability status (health check + UI display)

**Status:** ready-for-agent

- [ ] Control Pi API exposes an endpoint that sends a WoL magic packet to the server's MAC address, protected by the identity-header check from ticket 01
- [ ] Control Pi API asserts it is not bound off-tailnet (using ticket 01's helper)
- [ ] Tests exercise the HTTP boundary via Flask's test client, mocking only the raw WoL socket send
- [ ] Control UI's Wake button is disabled whenever the latest known state is Reachable, and enabled otherwise
- [ ] Pressing Wake calls the Control Pi API and does not implement its own no-op guard beyond what the UI's enable/disable state already provides — the API always attempts the action and reports the outcome
- [ ] Wake requested / succeeded / failed events are logged via the Grafana Alloy client from ticket 01, tagged with the caller's identity from the header
- [ ] Control Pi API and Control UI calls work same-origin on the Pi with no CORS entry needed
