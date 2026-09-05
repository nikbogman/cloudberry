# UI

The browser app served by the [Gateway](gateway.md): shows whether Compute is
[Reachable](../CONTEXT.md) and offers Wake/Suspend. The only user-facing surface
in the system.

A framework-light TypeScript + Vite SPA, kept minimal so it never needs to run
on the Pi Zero itself.

Code: [`ui/`](../ui).

## Behavior

All of it lives in [`mountUi`](../ui/src/app.ts):

- Polls the [Compute API](compute.md)'s `GET /health` every 12s (configurable),
  renders `Checking…` / `Reachable` / `Unreachable`.
- **Wake button** — enabled unless already reachable. `POST`s to the
  [Gateway API](gateway.md)'s `/wake` (same-origin by default).
- **Suspend button** — enabled unless already unreachable. `POST`s to the
  Compute API's `/suspend` (a distinct origin).
- Both actions are fire-and-forget: the outcome isn't surfaced in the UI.

## Configuration

Read at build/dev time via Vite env vars (see
[`main.ts`](../ui/src/main.ts)):

| Variable | Purpose |
|---|---|
| `VITE_COMPUTE_API_URL` | Compute API origin (default `''`, relative). |
| `VITE_GATEWAY_API_URL` | Gateway API origin (default `''`, relative — works in production since it's same-origin with the UI). |

## Deployment

Built on the dev machine (`npm run build`); only the static `dist/` output ships
to the Pi Zero, which never runs a Node toolchain. The Gateway binary
`//go:embed`s it at build time, so there is no separate UI artifact on the
device. Automated by [`deploy/deploy_gateway.py`](../deploy/deploy_gateway.py) —
the UI build must run before the Go build, which embeds whatever is in `uidist/`
at that moment.
