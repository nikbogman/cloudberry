# UI

The browser app served by the [Gateway](gateway.md): shows whether Compute is
[Reachable](../CONTEXT.md) and offers Wake/Suspend. The only user-facing surface
in the system.

Plain static files — HTML, one ES module, one stylesheet. No framework, no
bundler, no build step, no Node toolchain.

Code: [`ui/`](../ui).

## Behavior

All of it lives in [`mountUi`](../ui/app.js), called by the inline module script
in [`index.html`](../ui/index.html):

- Polls the [Compute API](compute.md)'s `GET /health` every 12s (configurable),
  renders `Checking…` / `Reachable` / `Unreachable`.
- **Wake button** — enabled unless already reachable. `POST`s to the
  [Gateway API](gateway.md)'s `/wake` (same-origin by default).
- **Suspend button** — enabled unless already unreachable. `POST`s to the
  Compute API's `/suspend` (a distinct origin).
- Failures surface in an error line; successes don't.

## Configuration

One value, `window.COMPUTE_API_URL` — the Compute API's origin. `index.html`
loads it from `/config.js`, which the [Gateway](gateway.md#routes) *serves* (from
its `COMPUTE_API_URL` env var) rather than the UI shipping. That keeps `ui/` a
pure static tree with nothing generated into it at deploy time.

Under a plain dev server `/config.js` 404s, the global stays undefined, and the
UI falls back to `''` — a relative path. The 404 in the console is expected.

The Gateway API origin is always relative — the Gateway serves both the UI and
`/wake`, so they're same-origin.

## Layout

```
index.html        markup + the inline mount script
app.js            all the behavior
style.css
icons/            favicons + touch icons, referenced by index.html
site.webmanifest  stays at the root; its icon paths point into icons/
ui.go             the go:embed declaration -- six lines, no logic
```

## Development

```sh
./dev-ui.sh          # http://localhost:5173, or ./dev-ui.sh <port>
```

Serves `ui/` straight from disk, so edits show up on refresh with no rebuild.
It writes `ui/config.js` (gitignored) from the repo root `.env`'s
`COMPUTE_TAILNET_HOST`, the same value a Deploy uses — so the health poll hits
the real Compute host and the status is live.

Wake is a Gateway route and 404s here. For it, run the real binary instead (see
the README) — it serves the *embedded* UI, so edits need a restart.

Absolute asset paths (`/app.js`, `/style.css`) mean the site must be served from
its own root, either way. Opening `index.html` as a `file://` URL will not work
— ES modules need an HTTP origin.

## Deployment

[`ui/ui.go`](../ui/ui.go) `//go:embed`s this directory straight into the Gateway
binary — no copy, no staging directory. The declaration sits *inside* `ui/`
because `go:embed` patterns are relative to their own package directory and
can't climb out of it: `cmd/gateway-api` cannot reach `../../ui`, but a package
living here can embed its own contents. Its globs skip `ui.go` itself.

[`deploy_gateway.py`](../deploy/deploy_gateway.py) just cross-compiles and ships
that one binary; nothing else reaches the Pi Zero.
