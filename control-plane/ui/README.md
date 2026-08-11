# UI

The browser app on the Pi Zero: shows whether the compute host is [Reachable](../../CONTEXT.md) and offers Wake/Suspend actions. The only user-facing surface in the system.

A framework-light TypeScript + Vite SPA, kept minimal so it never needs to run on the Pi Zero itself — see Deployment below.

## Behavior

All of it lives in [`mountUi`](src/app.ts):

- Polls the [Compute API](../README.md#compute-api)'s `GET /health` every 12s (configurable), renders `Checking…` / `Reachable` / `Unreachable`.
- **Wake button** — enabled unless already reachable. `POST`s to the [Gateway API](../README.md#gateway-api)'s `/wake` (same-origin by default).
- **Suspend button** — enabled unless already unreachable. `POST`s to the Compute API's `/suspend` (a distinct origin).
- Both actions are fire-and-forget: the outcome isn't surfaced in the UI.

## Configuration

Read at build/dev time via Vite env vars (see [`main.ts`](src/main.ts)):

| Variable | Purpose |
|---|---|
| `VITE_COMPUTE_API_URL` | Compute API origin (default `''`, relative). |
| `VITE_GATEWAY_API_URL` | Gateway API origin (default `''`, relative — works in production since it's same-origin with the UI). |

## Development

```sh
npm install
npm run dev         # Vite dev server
npm run test         # vitest
npm run typecheck    # tsc --noEmit
npm run build         # tsc && vite build -> dist/
```

## Deployment

Built on the dev machine (`npm run build`); only the static `dist/` output ships to the Pi Zero, which never runs a Node toolchain. Automated by [`provisioning/deploy_gateway.py`](../../provisioning/deploy_gateway.py) — see [`provisioning/README.md`](../../provisioning/README.md).
