# UI

The browser app served by [Edge](edge.md) at `/ui/`: shows whether Sleeper is
[Reachable](../CONTEXT.md) and offers Wake/Suspend. The only user-facing surface
in the system.

Plain static files — HTML, one ES module, one stylesheet. No framework, no
bundler, no build step, no Node toolchain.

Code: [`ui/`](../ui).

## Behavior

All of it lives in [`mountUi`](../ui/app.js), called by the inline module script
in [`index.html`](../ui/index.html):

- Polls `/api/sleeper/health` every 12s (configurable), renders
  `Checking…` / `Reachable` / `Unreachable`.
- **Wake button** — enabled unless already reachable. `POST`s to
  `/api/waker/wake`.
- **Suspend button** — enabled unless already unreachable. `POST`s to
  `/api/sleeper/suspend`.
- Failures surface in an error line; successes don't.

## Configuration

None. Everything is same-origin with Edge, so `index.html` passes the fixed
`/api/sleeper` and `/api/waker` base paths to `mountUi`, and Edge forwards them
to the [Sleeper API](sleeper.md) and [Waker API](waker.md).

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

Run Edge locally (see [edge.md](edge.md#development)) and open
`http://localhost:8080/ui/`. It serves the *embedded* UI, so edits need a
restart.

Asset paths are relative so the app works under `/ui/`. Opening `index.html` as
a `file://` URL will not work — ES modules need an HTTP origin.

## Deployment

[`ui/ui.go`](../ui/ui.go) `//go:embed`s this directory straight into the Edge
binary — no copy, no staging directory. The declaration sits *inside* `ui/`
because `go:embed` patterns are relative to their own package directory and
can't climb out of it: `cmd/edge-api` cannot reach `../../ui`, but a package
living here can embed its own contents. Its globs skip `ui.go` itself.
