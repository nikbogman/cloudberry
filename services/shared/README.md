# control-plane-shared

Shared helpers for the two Flask apps in this repo: [Pi API](../pi-api) and [Server API](../server-api). Not a standalone service — a library dependency, pulled in via an editable path dependency in each app's `pyproject.toml`.

See the repo-root [CONTEXT.md](../../CONTEXT.md) for the domain vocabulary these modules exist to enforce, and [docs/adr/](../../docs/adr/) for the decisions referenced below.

## Modules

- **`auth.py`** — `require_tailnet_identity`, a route decorator that rejects any request missing the `Tailscale-User-Login` header (the [Identity header](../../CONTEXT.md)) with a 401, and `get_caller_identity()` to read it back inside the view. Per [ADR-0004](../../docs/adr/0004-tailnet-membership-authorization.md), presence of the header is the entire authorization check — no allow-list of specific identities.
- **`bind_safety.py`** — `assert_tailnet_only_bind(host)`, called at app startup, raises `BindOffTailnetError` unless `host` is loopback or within Tailscale's own address ranges. A structural backstop for the "never reachable off-tailnet" requirement, independent of whatever fronts the app.
- **`alloy.py`** — `AlloyLogger.send_event(...)`, ships structured wake/suspend/reachability events to Grafana Alloy. Both the Pi API and Server API are stateless, so Alloy is the only place this history lives. Transport failures are logged and swallowed — a logging outage must never block the caller's actual action.

## Development

```sh
uv sync --extra dev
uv run pytest
uv run mypy src
```
