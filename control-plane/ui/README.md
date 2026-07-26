# UI

The browser-facing app on the Pi Zero: shows whether the compute host is [Reachable](../../CONTEXT.md) and offers Wake/Suspend actions. The only user-facing surface in the system — see the [UI](../../CONTEXT.md) entry in CONTEXT.md.

A framework-light TypeScript + Vite SPA (no framework), deliberately kept minimal so it never needs to run on the Pi Zero itself — see Deployment below.

## Behavior

All of it lives in [`mountUi`](src/app.ts):

- Polls the [Compute API](../compute-api)'s `GET /health` every 12s (configurable) and renders `Checking…` / `Reachable` / `Unreachable`.
- **Wake button** — enabled unless already reachable. `POST`s to the [Gateway API](../gateway-api)'s `/wake` (same-origin by default — the UI and Gateway API are served from the same `tailscale serve` app, [ADR-0001](../../docs/adr/0001-direct-browser-to-api-no-relay.md)).
- **Suspend button** — enabled unless already unreachable. `POST`s to the Compute API's `/suspend` (a distinct origin, per the same ADR).
- Both actions are fire-and-forget: the request outcome is not surfaced in the UI.

## Configuration

Read at build/dev time via Vite env vars (see [`main.ts`](src/main.ts)):

| Variable | Required | Purpose |
|---|---|---|
| `VITE_COMPUTE_API_URL` | no (default `''`, relative) | Compute API origin. |
| `VITE_GATEWAY_API_URL` | no (default `''`, relative) | Gateway API origin — relative works in production since it's same-origin with the UI. |

## Development

```sh
npm install
npm run dev         # Vite dev server
npm run test         # vitest
npm run typecheck    # tsc --noEmit
npm run build         # tsc && vite build -> dist/
```

## Deployment

Built on the dev machine (`npm run build`) and only the static `dist/` output is shipped to the Pi Zero — the Pi never runs a Node toolchain, a deliberate constraint given it's a single-core, low-memory device ([ADR-0006](../../docs/adr/0006-git-pull-deploy-with-offdevice-ui-build.md)). Automated by pyinfra's `deploy_gateway_api.py` (`../../pyinfra/deploy_gateway_api.py`, per `docs/specs/pyinfra-provisioning/spec.md`), which builds the UI on the dev machine and syncs `dist/` to the Pi — see [`pyinfra/README.md`](../../pyinfra/README.md).
