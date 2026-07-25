"""Deploys the Gateway API to `gateway`, plus delivers the UI's static build.

Cross-compiles the Gateway API's Go binary on the dev machine and ships
only the compiled binary to `gateway` -- the Pi Zero W never runs a Go
toolchain (ADR-0013, extended to this app by ADR-0015). Also builds the UI
on the dev machine (`npm ci && npm run build`) and syncs only its static
`dist/` to `gateway` -- the Gateway never runs a Node/npm toolchain
either. Targetable in isolation:

    pyinfra inventory.py deploy_gateway_api.py --limit gateway
    pyinfra inventory.py deploy_gateway_api.py --limit gateway --dry

`COMPUTE_API_ORIGIN` (baked into the UI build as `VITE_COMPUTE_API_URL`) is
no longer a separate required secret -- it's derived from
`InventorySettings.compute_tailnet_host` (the compute host's real Tailscale
MagicDNS name, distinct from `compute_host`, which is just pyinfra's SSH
target), since that's exactly the origin the Compute API's own
`tailscale serve` instance (deploy_compute_api.py) exposes it at. The UI,
not this Gateway's Caddy, routes to the Compute API (ADR-0001 rules out a
Gateway-side relay).

Required dev-machine env vars (fail fast if missing, only when targeting
`gateway`/its `test` stand-in): `COMPUTE_MAC_ADDRESS` and the
`GRAFANA_CLOUD_LOKI_*` trio (the Gateway API's own config).

Note on `--dry`: the UI build (`local.shell` below) and the Go binary
build (inside `go_binary_systemd_service`) always run, even under `--dry`
-- pyinfra's dry-run guarantee only covers remote operations, not a local
build step. Neither touches `gateway`, but isn't a true no-op.
"""

from pyinfra import local
from pyinfra.operations import files

from common import has_device_role
from go_deploy import go_binary_systemd_service
from settings import GatewayApiSecrets, GatewayApiSettings, InventorySettings

api_settings = GatewayApiSettings()
inventory = InventorySettings()

if has_device_role("gateway"):
    secrets = GatewayApiSecrets()

    # Pi Zero W is single-core ARM1176 (armv6) -- GOARM=6. A Pi Zero 2 W
    # would need GOARCH=arm64 (or GOARCH=arm GOARM=7 on a 32-bit OS)
    # instead; update this if the real device ever changes (same note as
    # deploy_caddy.py's wake_plugin build).
    go_binary_systemd_service(
        module_dir="../services",
        package="./cmd/gateway-api",
        goos="linux",
        goarch="arm",
        goarm="6",
        binary_name="gateway-api",
        remote_binary="/usr/local/bin/gateway-api",
        unit_name="gateway-api.service",
        description="Gateway API -- sends Wake-on-LAN to the compute host",
        environment={
            "COMPUTE_MAC_ADDRESS": secrets.compute_mac_address,
            "GATEWAY_API_HOST": api_settings.gateway_api_host,
            "GATEWAY_API_PORT": str(api_settings.gateway_api_port),
            "GRAFANA_CLOUD_LOKI_URL": secrets.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": secrets.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": secrets.grafana_cloud_loki_api_key,
        },
    )

    # VITE_COMPUTE_API_URL bakes in the real origin so the browser calls the
    # Compute API directly (ADR-0001). VITE_GATEWAY_API_URL is left unset --
    # same-origin works since the Gateway API shares this Caddy site.
    local.shell(
        f"cd ../services/ui && npm ci && "
        f"VITE_COMPUTE_API_URL=https://{inventory.compute_tailnet_host} npm run build",
        print_output=True,
    )

    files.sync(
        name="Sync the UI's static build to gateway",
        src="../services/ui/dist",
        dest="/srv/ui",
        delete=True,
    )
