# Control UI

The browser-facing app on the Pi Zero: shows whether the main server is [Reachable](../../CONTEXT.md) and offers Wake/Suspend actions. The only user-facing surface in the system — see the [Control UI](../../CONTEXT.md) entry in CONTEXT.md.

A framework-light TypeScript + Vite SPA (no framework), deliberately kept minimal so it never needs to run on the Pi Zero itself — see Deployment below.

## Behavior

All of it lives in [`mountControlUi`](src/app.ts):

- Polls the [Control server API](../control_server_api)'s `GET /health` every 12s (configurable) and renders `Checking…` / `Reachable` / `Unreachable`.
- **Wake button** — enabled unless already reachable. `POST`s to the [Control Pi API](../control_pi_api)'s `/wake` (same-origin by default — the Control UI and Control Pi API are served from the same `tailscale serve` app, [ADR-0001](../../docs/adr/0001-direct-browser-to-api-no-relay.md)).
- **Suspend button** — enabled unless already unreachable. `POST`s to the Control server API's `/suspend` (a distinct origin, per the same ADR).
- Both actions are fire-and-forget: the request outcome is not surfaced in the UI, only in the Alloy audit log each API writes.

## Configuration

Read at build/dev time via Vite env vars (see [`main.ts`](src/main.ts)):

| Variable | Required | Purpose |
|---|---|---|
| `VITE_CONTROL_SERVER_API_URL` | no (default `''`, relative) | Control server API origin. |
| `VITE_CONTROL_PI_API_URL` | no (default `''`, relative) | Control Pi API origin — relative works in production since it's same-origin with the UI. |

## Development

```sh
npm install
npm run dev         # Vite dev server
npm run test         # vitest
npm run typecheck    # tsc --noEmit
npm run build         # tsc && vite build -> dist/
```

## Deployment

Built on the dev machine (`npm run build`) and only the static `dist/` output is shipped to the Pi Zero — the Pi never runs a Node toolchain, a deliberate constraint given it's a single-core, low-memory device ([ADR-0006](../../docs/adr/0006-git-pull-deploy-with-offdevice-ui-build.md)). Automated by pyinfra's `deploy_control_pi_api.py` (`../../pyinfra/deploy_control_pi_api.py`, per `.scratch/pyinfra-provisioning/spec.md`), which builds the UI on the dev machine and syncs `dist/` to the Pi — see [`pyinfra/README.md`](../../pyinfra/README.md).
