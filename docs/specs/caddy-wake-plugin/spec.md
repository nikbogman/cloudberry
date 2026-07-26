Status: ready-for-agent

# Custom Caddy wake plugin (replaces caddy-wol)

## Problem Statement

The auto-wake proxy currently depends on a third-party Caddy plugin, `github.com/dulli/caddy-wol`, to build and broadcast its own Wake-on-LAN packet whenever a proxied request finds the main server asleep. This means WoL-sending logic exists in two independently-maintained places — the third-party plugin and the Control Pi API's own `wol.py` — with no guarantee they stay behaviorally consistent, and the auto-wake proxy's Caddy binary depends on code this repo doesn't own or audit.

## Solution

Replace `caddy-wol` with a small, custom, in-repo Caddy plugin that triggers a wake by calling the Control Pi API's existing `POST /wake` endpoint over loopback HTTP, instead of sending a WoL packet itself. This collapses WoL-sending to one implementation and makes the auto-wake proxy a second caller of the same wake action the Control UI's Wake button already uses — a deliberate, documented reversal of ADR-0005's "independent wake paths" framing, recorded in ADR-0012. See `CONTEXT.md`'s updated Auto-wake proxy, Wake, and Identity header entries for the resulting vocabulary, and ADR-0004's amendment for the accompanying loopback-auth exception.

## User Stories

1. As the operator, I want the auto-wake proxy to trigger a wake by calling the Control Pi API's existing wake action, so that WoL-sending logic exists in exactly one place instead of two independently-maintained ones.
2. As the operator, I want the third-party `dulli/caddy-wol` dependency removed, so that the auto-wake proxy's Caddy binary only depends on code this repo owns and can audit.
3. As the operator, I want the new plugin to throttle repeat wake calls per upstream to about once every 10 minutes, so that a burst of held/retried requests during a wake cycle doesn't hammer the Control Pi API.
4. As the operator, I want the plugin to proceed with its hold/retry of the proxied request even if the call to the Control Pi API fails or times out, so that a transient Control Pi API hiccup doesn't turn into an immediate error for the caller.
5. As the operator, I want the Control Pi API's wake action to accept a caller on the Pi's own loopback address, so that the auto-wake proxy can trigger it without needing a Tailscale identity of its own.
6. As the operator, I want the existing Tailscale-identity-authenticated path to `/wake` (used by the Control UI's Wake button) to keep working exactly as before, so that this change doesn't weaken or alter the deliberate Wake action.
7. As the operator, I want `SERVER_MAC_ADDRESS` and `WOL_BROADCAST_ADDRESS` removed from Caddy's own environment and Caddyfile, so that MAC-address/broadcast-address knowledge lives in exactly one place (the Control Pi API's existing secret).
8. As the operator, I want the new Caddyfile directive (`call_wake_api <url>`) to read the Control Pi API's URL from the same host/port settings pyinfra already templates elsewhere in the Caddyfile, so that there's no new source of truth for that address.
9. As the operator, I want `deploy_caddy.py`'s docstring, `services/auto_wake_proxy/README.md`, `pyinfra/README.md`, and `docs/agents/pyinfra-demo.md` updated to describe building Caddy with the new in-repo module instead of `github.com/dulli/caddy-wol`, so that the documented build instructions match what's actually required.
10. As the operator, I want the superseded `services/auto_wake_proxy/Caddyfile` reference file and its README's design-rationale sections updated to describe the `call_wake_api` mechanism, so that the kept design-reference record doesn't describe a mechanism no longer in use.
11. As the operator, I want building the Caddy binary with the new module to remain a manual prerequisite (not built by pyinfra itself), so that this stays consistent with the already-documented gap around Caddy binary provisioning.
12. As the operator, I want the plugin's throttling and fail-open behavior covered by Go unit tests against a fake Control Pi API server, so that this behavior is verified without needing a real device.
13. As the operator, I want the rendered Caddyfile validated with a real `caddy validate` run against a binary built with the new module, so that the directive's syntax and wiring are confirmed correct, not merely assumed.
14. As the operator, I want the loopback exception on `/wake` covered by tests using the existing Flask test-client fixtures, so that both the loopback path and the still-enforced identity-header path are verified together.
15. As the operator, I want a caller reaching `/wake` from any address other than `127.0.0.1`, without the identity header, to still receive a 401, so that the loopback exception doesn't widen who can trigger a wake beyond the Pi itself.
16. As the operator, I want the plugin's throttle window keyed per upstream URL, so that a wake cycle for one workload doesn't suppress a wake call for a different, independently-asleep upstream, even though today there is only one upstream (ADR-0011).
17. As the operator, I want the plugin to log the outcome of its call to `/wake` (sent/succeeded, sent/failed, or skipped/throttled), so that operators can distinguish these cases from Caddy's own logs.

## Implementation Decisions

**Plugin module and directive**: a new Go module at `services/auto_wake_proxy/wake_plugin/`, registered as a Caddy HTTP handler module exposing a new Caddyfile directive `call_wake_api <url>`, taking one argument — the full URL of the Control Pi API's `/wake` endpoint. Replaces the `wake_on_lan {$SERVER_MAC_ADDRESS} {$WOL_BROADCAST_ADDRESS:255.255.255.255:9}` line inside the `auto_wake_route` snippet in `pyinfra/templates/Caddyfile.j2`, used as `call_wake_api http://127.0.0.1:{$CONTROL_PI_API_PORT}/wake` — reusing the `control_pi_api_port` value already templated into the Caddyfile for the existing `/wake*` UI route, so there's no new source of truth for that address. The existing `order wake_on_lan before respond` → `order call_wake_api before respond` wiring and the `handle_errors` / `@server_asleep expression {err.status_code} == 502` gating stay as-is — this only changes what the handler *does*, not when it's triggered.

**Handler behavior**: on invocation, check an in-memory, per-upstream-URL last-called timestamp. If a call went out within the throttle window (~10 minutes), skip the HTTP call, log "throttled," and return immediately so Caddy proceeds straight to the existing hold/retry `reverse_proxy`. Otherwise, POST to the configured URL with a short timeout (a few seconds — this is fire-and-forget relative to the hold/retry, not itself the hold/retry window), log the outcome (sent/succeeded, sent/failed, or timed out), record the call time regardless of outcome, and always let Caddy continue to the hold/retry `reverse_proxy` afterward (fail-open). Throttle state is process-local and in-memory — matches `caddy-wol`'s own in-process throttling, and is sufficient since there is exactly one Caddy process per Pi.

**`control_pi_api` `/wake` auth**: `require_tailnet_identity` becomes "identity header OR loopback caller" — pass if either the existing `Tailscale-User-Login` header check succeeds, or the request's remote address is `127.0.0.1`. When the loopback path is taken, the identity recorded for Alloy logging (`wake_requested`/`wake_succeeded`/`wake_failed`) is a fixed, non-tailnet string (e.g. `"auto-wake-proxy"`) distinguishing it from a real caller identity, so events stay attributable to their actual trigger.

**Caddyfile/pyinfra cleanup**: `pyinfra/templates/Caddyfile.j2` drops all `{$SERVER_MAC_ADDRESS}`/`{$WOL_BROADCAST_ADDRESS}` references. `pyinfra/deploy_caddy.py`'s docstring, `pyinfra/README.md`, `services/auto_wake_proxy/README.md`, and `docs/agents/pyinfra-demo.md` are updated to describe building Caddy with the new in-repo module (`services/auto_wake_proxy/wake_plugin/`) in place of `github.com/dulli/caddy-wol` — still a manual prerequisite, not built by pyinfra itself, consistent with the already-documented "Caddy binary provisioning" gap.

**Design-reference file**: `services/auto_wake_proxy/Caddyfile` (kept only as the superseded design reference, per its own header comment) and its README's plugin-research/rationale sections are updated to describe `call_wake_api` in place of `wake_on_lan`/`caddy-wol`, so the kept reference doesn't describe a mechanism no longer in use. `docs/specs/pyinfra-provisioning/issues/07-caddy.md` is left untouched — a closed historical record.

## Testing Decisions

A good test here exercises the plugin's/endpoint's externally observable behavior — what gets called, what gets returned, what gets logged — never internal state, matching this repo's existing "mock only the true external boundary" philosophy (e.g. `WakeOnLanSender` and `AlloyLogger` are mocked at the app boundary in `services/control_pi_api/tests/test_app.py`, not the socket call inside them).

- **Go plugin**: unit tests in `services/auto_wake_proxy/wake_plugin/` against the handler function directly, using `net/http/httptest.Server` as the fake Control Pi API. Cover: a request is sent on invocation; throttling suppresses a second call within the window for the same upstream URL; a different upstream URL is unaffected by another URL's throttle; the handler always signals "continue" regardless of whether the fake server returns success, an error status, or the request times out. No existing Go test prior art in this repo (greenfield for Go) — this establishes the pattern.
- **Caddyfile template**: reuse the exact validation seam already established in `docs/specs/pyinfra-provisioning/issues/07-caddy.md` and `pyinfra/README.md`'s three-tier testing procedure — render `Caddyfile.j2` and run a real `caddy validate` against the output using a binary built with the new module, confirming `call_wake_api` parses and every other directive remains valid.
- **`control_pi_api` `/wake`**: extend `services/control_pi_api/tests/test_app.py` with the existing fixtures (`client`, `alloy`, `wol_sender`). Add cases: (a) a request with no identity header but `environ_base={"REMOTE_ADDR": "127.0.0.1"}` succeeds and sends the magic packet; (b) a request from a non-loopback address with no identity header still returns 401, guarding against the exception accidentally widening; (c) the existing identity-header path from a non-loopback address is unaffected.

## Out of Scope

- Building/publishing the Caddy binary itself (with the new module compiled in) as part of pyinfra's Deploy — remains a manual prerequisite, same as it was for `caddy-wol`.
- Any change to the deliberate Wake button / Control UI flow — this feature only changes how the auto-wake proxy triggers the existing wake action, not the action itself.
- Any change to Suspend, reachability health checks, or the Control server API.
- A shared/external throttle store — in-process throttling is sufficient since there's exactly one Caddy process per Pi.
- Widening the loopback exception beyond `/wake` — no other Control Pi API endpoints currently exist, and none are added by this change.

## Further Notes

- ADR-0012 (supersedes ADR-0005's "independent wake paths" framing), the ADR-0004 amendment (loopback exception), and `CONTEXT.md`'s updated Auto-wake proxy/Wake/Identity header entries are already written as of this spec — see `docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md` and `docs/adr/0004-tailnet-membership-authorization.md`. Implementation should treat these as settled context, not open questions.
- Domain vocabulary: use "auto-wake proxy," "Control Pi API," "wake," and "identity header" exactly as defined in `CONTEXT.md`.
- This spec assumes `services/control_pi_api` and `pyinfra/deploy_caddy.py` as already built per `docs/specs/homelab-control-system/spec.md` and `docs/specs/pyinfra-provisioning/spec.md` respectively — this feature modifies both in place rather than building either from scratch.
