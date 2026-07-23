# wake_plugin

A custom Caddy HTTP handler module registering the `call_wake_api` Caddyfile directive, used by [`../../../pyinfra/templates/Caddyfile.j2`](../../../pyinfra/templates/Caddyfile.j2) to trigger a wake. Instead of building and broadcasting its own Wake-on-LAN packet, it POSTs to the Pi API's `/wake` — see [ADR-0012](../../../docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md) for why.

## Directive

```
call_wake_api <url>
```

On invocation, POSTs to `<url>` (the Pi API's `/wake`, e.g. `http://127.0.0.1:5000/wake`), then always calls the next handler in the chain regardless of that call's outcome — the caller always proceeds to the hold/retry `reverse_proxy` that follows it in `Caddyfile.j2`. Repeat calls to the same URL are throttled to at most one per ~10 minutes.

Must be ordered before `respond` (`order call_wake_api before respond` as a global option).

## Development

```sh
go test ./...
go vet ./...
```

## Building a Caddy binary with this module

Via [`xcaddy`](https://github.com/caddyserver/xcaddy). `pyinfra/deploy_caddy.py` now builds and ships this binary automatically (cross-compiled for the Pi Zero W, ADR-0013) — the command below is the same build run by hand, e.g. for local testing:

```sh
xcaddy build --with github.com/nikbogman/homelab/services/pi-proxy/wake_plugin=./services/pi-proxy/wake_plugin
```

The `=./services/...` local override is required until this repo is fetchable as a public Go module — without it, `xcaddy` would try to `go get` the module path from its VCS host. Add `GOOS=linux GOARCH=arm GOARM=6` (as `deploy_caddy.py` does) to cross-compile for the Pi Zero W instead of the host machine.

## Verification performed

- `go test ./...` — throttling, different-URL isolation, and fail-open behavior (an error response or a timeout from the wake URL doesn't block the caller) against a `net/http/httptest.Server` standing in for the Pi API.
- `caddy validate` against `Caddyfile.j2` rendered with representative values, using a binary built with this module — reports `Valid configuration`.
- A live `caddy run` smoke test against that config, a fake Pi API, and a deliberately unreachable upstream: the first request to `/server*` fires a real `POST /wake` and returns 502 once the retry window elapses; a second request within the throttle window skips the `/wake` call ("throttled, skipping") and still returns 502 the same way.

See `docs/agents/pyinfra-demo.md`'s "caddy-wake-plugin" section for the full write-up.
