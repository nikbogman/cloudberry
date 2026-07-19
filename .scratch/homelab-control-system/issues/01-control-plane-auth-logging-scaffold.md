# 01 — Control-plane auth & logging scaffold

**What to build:** A shared, tested helper that both the Control Pi API and Control server API will adopt: it rejects any request that doesn't carry a valid `Tailscale-User-Login` identity header, asserts the app process is bound only to loopback/tailnet interfaces (never off-tailnet — a hard requirement per ADR-0004), and provides a Grafana Alloy logging client for shipping structured events (action, outcome, caller identity). This ticket delivers the scaffold itself, verified against a minimal Flask app — not a full production endpoint yet.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] A decorator/middleware rejects requests missing or carrying an invalid `Tailscale-User-Login` header, tested at the HTTP boundary with Flask's test client
- [ ] A reusable assertion (used at app startup) confirms the process is not bound to any off-tailnet address, and fails loudly if it is
- [ ] A Grafana Alloy logging client can ship a structured event (event type, outcome, caller identity) — the actual Alloy transport is the one seam that may be mocked in tests
- [ ] Both helpers are documented well enough that tickets 02–04 can import and use them without re-deriving the auth/logging pattern
