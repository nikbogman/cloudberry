# UI

The browser app served by [Edge](../cmd/edge/README.md) at `/ui/`: shows whether blackberry is
[Reachable](../CONTEXT.md) and offers Wake/Suspend. The only user-facing surface
in the system.

Plain static files — HTML, one ES module, one stylesheet. No framework, no
bundler, no build step, no Node toolchain.

Code: [`ui/`](.).

## Behavior

All of it lives in [`mountUi`](app.js), called by the inline module script
in [`index.html`](index.html):

- Polls `/api/hostd/health` every 12s (configurable), renders
  `Checking…` / `Reachable` / `Unreachable`.
- **Wake button** — enabled unless already reachable. `POST`s to
  `/api/waker/wake`.
- **Suspend button** — enabled unless already unreachable. `POST`s to
  `/api/hostd/suspend`.
- Failures surface in an error line; successes don't.

## Configuration

None. Everything is same-origin with Edge, so `index.html` passes the fixed
`/api/hostd` and `/api/waker` base paths to `mountUi`, and Edge forwards them
to [hostd](../cmd/hostd/README.md) and [waker](../cmd/waker/README.md).

## Development

Run Edge locally (see [edge](../cmd/edge/README.md#development)) and open
`http://localhost:8080/ui/`. It serves the *embedded* UI, so edits need a
restart.

Asset paths are relative so the app works under `/ui/`. Opening `index.html` as
a `file://` URL will not work — ES modules need an HTTP origin.

## Deployment

[`ui/ui.go`](ui.go) `//go:embed`s this directory into the edge binary;
its comment explains why it lives inside `ui/`.
