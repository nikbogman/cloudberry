# Auto-wake proxy

Caddy configuration for the "Auto-wake proxy" component of the homelab
control system (`.scratch/homelab-control-system/spec.md`, ticket
`.scratch/homelab-control-system/issues/05-auto-wake-proxy.md`, ADR-0005).
Unlike `services/shared`, `services/control_pi_api`, `services/control_server_api`,
and `services/control_ui`, this is not a Python/TypeScript application —
it's a declarative Caddy config artifact. There is one file:

- `Caddyfile` — the source of truth, heavily commented in place. Each
  section maps directly to one of the ticket's four acceptance criteria;
  see the comments above the `(auto_wake_route)` snippet for that mapping.

## Plugins (real, researched — not invented)

This Caddyfile requires a Caddy binary built with one third-party plugin
compiled in (Caddy plugins are Go modules linked in at build time via
`xcaddy` — there's no dynamic-loading mechanism to add them at runtime). A
second plugin was researched per this ticket's instructions but isn't
needed by anything this file actually does — see item 2 below.

1. **[`github.com/dulli/caddy-wol`](https://github.com/dulli/caddy-wol)**
   — provides the `wake_on_lan` HTTP handler directive. Chosen because its
   documented pattern (`reverse_proxy` fails → `handle_errors` catches the
   502 → `wake_on_lan` sends the magic packet → a second `reverse_proxy`
   retries with a longer `lb_try_duration`) matches the ticket's "sends the
   WoL packet and holds/retries the proxied request until the server
   responds" requirement precisely: the *original* client request is held
   and retried server-side, and the client sees a normal successful
   response once the backend wakes.

   Alternative found and rejected: **[`github.com/Stoufiler/caddy-jellywol`](https://github.com/Stoufiler/caddy-jellywol)**.
   It's also a real, working WoL-triggering Caddy plugin, but its
   documented behavior returns `503 Service Unavailable` with a
   `Retry-After` header and depends on the *client* re-requesting after
   the delay. That's a reasonable design for its target use case (media
   players like Infuse/Jellyfin that understand `Retry-After`), but it
   doesn't give a general "succeeds transparently, does not fail outright"
   guarantee for arbitrary HTTP clients hitting Immich, an "AI agents" UI,
   etc. — which is what this ticket's acceptance criteria ask for. It's
   also narrowly built around health-checking a specific `ping_ip`/
   `ping_port`, a media-server-shaped design rather than a general-purpose
   reverse-proxy WoL handler.

2. **[`github.com/tailscale/caddy-tailscale`](https://github.com/tailscale/caddy-tailscale)**
   — Tailscale's own official Caddy plugin. Real and researched per the
   task's request to check for one. It provides `bind tailscale/<node>`
   (join the tailnet directly via embedded `tsnet`, no separate `tailscale
   serve` needed) and a `tailscale_auth` directive for asserting identity.
   **Not used for binding in this file, and not part of the recommended
   build below.** This repo's established convention (see `CONTEXT.md`'s
   Control UI entry, and the `bind_host` + `assert_tailnet_only_bind`
   pattern shared by `control_pi_api` and `control_server_api`) is: bind
   the app to loopback, and let an *external* `tailscale serve` instance
   expose it to the tailnet. No Control-UI-serving Caddy config exists in
   this repo yet — this ticket only adds the workload routes — but per
   CONTEXT.md that's the model it's expected to follow when it's built.
   Mixing in a second, different tailnet-joining mechanism just for these
   workload routes would pre-empt that future config's approach rather
   than match it. The `default_bind {$AUTO_WAKE_PROXY_BIND_HOST:127.0.0.1}`
   global option in the Caddyfile satisfies the same ADR-0004 "never bound
   off-tailnet" hard requirement the existing convention relies on. If a
   future ticket moves the whole Pi Caddy instance onto `tsnet` directly,
   `bind tailscale/<node>` is the real, correct directive for that.

   (Compiling this plugin in anyway "just in case" was considered and
   rejected — a build-time dependency nothing in the file exercises is
   unnecessary complexity for a capability no current ticket needs; see
   "Verification performed" below for confirmation the Caddyfile validates
   identically with or without it.)

Build command for the Caddy binary this Caddyfile actually needs (via
[`xcaddy`](https://github.com/caddyserver/xcaddy) — this is a build-time
concern; wiring it into an actual device install is pyinfra's future Caddy
Concern, out of this ticket's scope per `.scratch/pyinfra-provisioning/spec.md`):

```sh
xcaddy build --with github.com/dulli/caddy-wol
```

## Verification performed

No pytest/vitest seam applies to a Caddyfile, and there's no real Pi/server
to test against, so verification here means: is the syntax genuinely valid,
and does the mechanism genuinely behave as claimed. Both were checked for
real, not assumed:

1. **Two custom Caddy binaries were actually built** in the sandbox (Go
   1.23 + `xcaddy`, both installed fresh — no `go`/`caddy`/`xcaddy` were
   preinstalled): one with `caddy-wol` only (the recommended build), and
   one with `caddy-wol` + `caddy-tailscale` together, confirmed via
   `caddy list-modules` to register `http.handlers.wake_on_lan` (and, in
   the second binary, `tailscale`/`http.authentication.providers.tailscale`).
2. **`caddy validate --config Caddyfile --adapter caddyfile`** passes
   against the actual committed file (with placeholder env vars set) on
   *both* binaries — confirming the file is valid for the recommended
   single-plugin build, and that adding `caddy-tailscale` changes nothing
   about its validity (supporting the "not needed by this file" claim
   above with evidence, not just assertion).
3. **`caddy fmt --diff`** reports no formatting drift (the file was run
   through `caddy fmt --overwrite` once, and now matches Caddy's own
   canonical formatting).
4. **Live `caddy run` smoke tests** against the actual Caddyfile's logic
   (a copy with only the hold/retry window shortened for test speed, 120s →
   8s — no other change):
   - **Asleep → wakes mid-request → succeeds (AC2, AC3):** a mock backend
     that starts listening only several seconds after the request is
     fired, simulating a server waking from suspend mid-request. Backend
     down at request time → Caddy's log shows
     `"msg":"dispatched magic packet","mac":"00:11:22:33:44:55"` — the
     `wake_on_lan` handler fired for real, from the exact `handle_errors`
     path this file uses. The original client request was held and
     retried; once the mock backend started listening mid-flight, the
     *same* `curl` call completed with `HTTP_STATUS:200` and the mock's
     real response body — the request that arrived while "asleep"
     succeeded transparently rather than failing, in ~6s (matching the
     artificial 5s wake delay), not an error.
   - **Already-awake fast path:** hitting an already-up backend returns
     its response immediately, no WoL, no delay.
   - **Live backend returning its own genuine 502 does NOT trigger WoL:**
     a mock backend that's up and immediately answers with a real
     `502 genuine app-level 502 from a LIVE backend` body. The response
     was proxied straight through unmodified (`HTTP_STATUS:502`, the
     exact body) and the log had zero `"dispatched magic packet"` lines —
     confirming the `handle_errors`/`@server_asleep` matcher only catches
     Caddy's own fabricated "can't reach any backend" error, not a real
     HTTP 502 an actually-running app chose to send. This directly checks
     the concern that "server asleep" detection might false-positive on a
     live-but-erroring workload; it doesn't.
5. **Independence from the Control UI/Control Pi API (AC4)** verified by
   `grep` over the committed Caddyfile for any reference to those services
   (origins, ports, endpoint paths, env var names) — the only matches are
   explanatory comments; there is no functional directive (`reverse_proxy`
   target, `import`, header, etc.) anywhere in the file that touches either
   service. This is structural, not just descriptive: nothing in this file
   can fail to compile or run differently based on whether those services
   exist, are running, or are reachable.

The build toolchain (Go, `xcaddy`, the compiled binary) is scratch/verification
tooling only — not part of the deployed artifact, and not added to this repo.
