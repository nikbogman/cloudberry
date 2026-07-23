Status: ready-for-human

# 01 — Pre-existing bug found and fixed: `handle_errors` nested inside `handle_path`

**Not part of the original spec's scope.** Found while running the spec's own testing decision (`caddy validate` against the rendered `Caddyfile.j2` with a real binary) and fixed here because it blocked verifying the exact route this spec touches. Flagging for human review since it's a correction to existing repo state, not a new feature.

## What was found

`pyinfra/README.md`'s "What was actually verified" section already flagged that ADR-0011's `/server*` route (`handle_path /server* { import auto_wake_route ... }`, where `auto_wake_route` contains a nested `handle_errors`) was never actually run through a real `caddy validate` — only Jinja2-rendered and eyeballed, since no Caddy build toolchain was available when ADR-0011 was added.

Building a real binary (with `wake_plugin` compiled in, per this spec) and validating the rendered template surfaced a genuine Caddy limitation: `handle_errors` is only usable as a site-level directive. Nested inside `handle_path` (or any `handle`/`route` block), Caddy's Caddyfile adapter rejects it — first with `directive 'handle_errors' is not an ordered HTTP handler`, and, after wrapping in an explicit `route {}` block, with `handle_errors directive returned something other than an HTTP route or subroute (only handler directives can be used in routes)`. Confirmed directive-independent: the same error occurs with a stock `respond` directive in place of the wake directive, so this predates and is unrelated to ADR-0012's `caddy-wol` → `call_wake_api` swap.

## Fix

`handle_errors` moved to the top level of the site block in `pyinfra/templates/Caddyfile.j2`, scoped to `/server*` via `{http.request.orig_uri.path}.startsWith("/server")` (matching the original, pre-rewrite request path — `handle_path` has already stripped `/server` from the working path by the time an error propagates to a site-level `handle_errors`). The `auto_wake_route` snippet's reusability is dropped in the process since there's exactly one call site today (ADR-0011); reintroduce a snippet if a second workload route is ever added.

## Verification

- `caddy validate` against the rendered template: `Valid configuration`.
- Live `caddy run` smoke test: a request to `/server*` against a fake Control Pi API and a deliberately unreachable upstream correctly fires `POST /wake` and returns a real (502) response once the retry window elapses; a second request within the throttle window skips the `/wake` call and still returns the same way.

Full write-up: `docs/agents/pyinfra-demo.md`'s "caddy-wake-plugin" section.
