# Caddy removed; the Gateway absorbs its job

Caddy is removed from the Gateway entirely. `tailscale serve` already terminates the tailnet-only HTTPS boundary directly in front of whatever Caddy used to listen on, which left Caddy doing nothing Go's standard library doesn't already do: serving static files, a path-routed reverse proxy, and a retry loop. `gateway-api` is renamed to `gateway` and absorbs all three jobs — it now serves the UI's static build at `/`, sends Wake-on-LAN on `POST /wake` (unchanged), and reverse-proxies `/server*` to the compute host, auto-waking it on a transport failure.

We considered and rejected keeping Caddy around just for TLS termination — moot, since `tailscale serve` already owns that in front of it. We also considered keeping the existing two-stage retry (an immediate 3s hold, escalating to a 120s hold after the first `call_wake_api` fires, see ADR-0012) and rejected it in favor of a single-stage 60s/1s retry window: the first, shorter stage never actually bought anything once `tailscale serve` already gates who can reach this route at all — there's no untrusted-traffic case it was shielding against that the tailnet boundary doesn't already cover.

What's gained: one binary instead of two, one systemd unit instead of two, one Deploy file instead of two, and no more `xcaddy`/cross-compiled-Caddy-plugin build step on the dev machine.

Consequences:

- The 90s auto-wake throttle (ADR-0012) is now unkeyed — a single `lastCall time.Time`, not a per-URL map — since this service has exactly one compute upstream (ADR-0011's single blind proxy target), so there's only ever one throttle key to have.
- The manual `/wake` path's loopback-or-identity-header gate (ADR-0004, ADR-0012) is preserved unchanged, even though there's no longer a separate process making that loopback call — the endpoint itself, and its authorization rule, are unaffected by which process happens to call it.
