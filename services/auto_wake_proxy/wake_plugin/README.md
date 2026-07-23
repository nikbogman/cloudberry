# wake_plugin

A custom Caddy HTTP handler module registering the `call_wake_api` Caddyfile directive, used by [`../../../pyinfra/templates/Caddyfile.j2`](../../../pyinfra/templates/Caddyfile.j2) in place of the third-party [`caddy-wol`](https://github.com/dulli/caddy-wol) plugin's `wake_on_lan` directive. Instead of building and broadcasting its own Wake-on-LAN packet, it POSTs to the Control Pi API's `/wake` — see [ADR-0012](../../../docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md) for why.

## Directive

```
call_wake_api <url>
```

On invocation, POSTs to `<url>` (the Control Pi API's `/wake`, e.g. `http://127.0.0.1:5000/wake`), then always calls the next handler in the chain regardless of that call's outcome — the caller always proceeds to the hold/retry `reverse_proxy` that follows it in `Caddyfile.j2`. Repeat calls to the same URL are throttled to at most one per ~10 minutes, matching `caddy-wol`'s prior per-host throttling.

Must be ordered before `respond` (`order call_wake_api before respond` as a global option), same as `wake_on_lan` was.

## Development

```sh
go test ./...
go vet ./...
```

## Building a Caddy binary with this module

Via [`xcaddy`](https://github.com/caddyserver/xcaddy), same build-time-only concern as `caddy-wol` was — `pyinfra/deploy_caddy.py` templates the config but does not build/install this binary (see that file's docstring and `pyinfra/README.md`'s Known gaps):

```sh
xcaddy build --with github.com/nikbogman/homelab/services/auto_wake_proxy/wake_plugin=./services/auto_wake_proxy/wake_plugin
```

The `=./services/...` local override is required until this repo is fetchable as a public Go module — without it, `xcaddy` would try to `go get` the module path from its VCS host.

## Verification performed

- `go test ./...` — throttling, different-URL isolation, and fail-open behavior (an error response or a timeout from the wake URL doesn't block the caller) against a `net/http/httptest.Server` standing in for the Control Pi API.
- `caddy validate` against `Caddyfile.j2` rendered with representative values, using a binary built with this module — reports `Valid configuration`.
- A live `caddy run` smoke test against that config, a fake Control Pi API, and a deliberately unreachable upstream: the first request to `/server*` fires a real `POST /wake` and returns 502 once the retry window elapses; a second request within the throttle window skips the `/wake` call ("throttled, skipping") and still returns 502 the same way.

See `docs/agents/pyinfra-demo.md`'s "caddy-wake-plugin" section for the full write-up.
