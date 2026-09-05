"""Deploys the Gateway to `gateway`: builds the UI, embeds it into the
Gateway binary, and ships the one resulting binary. Neither a Node nor a
Go toolchain ever runs on the Pi Zero W. Targetable in isolation:

    pyinfra inventory.py deploy_gateway.py --limit gateway
    pyinfra inventory.py deploy_gateway.py --limit gateway --dry

`VITE_COMPUTE_API_URL` (baked into the UI build) is derived from
`InventorySettings.compute_tailnet_host`, the Compute API's real
Tailscale MagicDNS name -- not a separate secret.

This binary also serves the UI's static files and reverse-proxies
`/server*` to the compute host, so it's also the sole thing standing up
`tailscale serve` for this device.

Required env vars (only when targeting `gateway`/its `test` stand-in):
`COMPUTE_MAC_ADDRESS`, `COMPUTE_PROXY_PORT`, and the `GRAFANA_CLOUD_LOKI_*`
trio.

The UI build and the Go build both always run, even under `--dry` --
pyinfra's dry-run guarantee only covers remote operations. The UI build
must run first: the Go build embeds whatever's in `uidist/` at that
moment.
"""

from pyinfra import host, local
from pyinfra.operations import server

from common import TailscaleServeStatus, has_device_role
from go_build import go_binary_systemd_service
from settings import GatewaySecrets, GatewaySettings, GrafanaLokiSecrets, InventorySettings

gateway_settings = GatewaySettings()
inventory = InventorySettings()

if has_device_role("gateway"):
    secrets = GatewaySecrets()
    grafana = GrafanaLokiSecrets()

    # VITE_GATEWAY_API_URL is left unset -- same-origin works since the
    # Gateway serves both the UI and /wake.
    local.shell(
        f"cd ../ui && npm ci && "
        f"VITE_COMPUTE_API_URL=https://{inventory.compute_tailnet_host} npm run build",
        print_output=True,
    )

    # Pi Zero W is ARM1176 (armv6) -- GOARM=6. A Pi Zero 2 W would need
    # GOARCH=arm64 instead.
    go_binary_systemd_service(
        module_dir="..",
        package="./cmd/gateway-api",
        goos="linux",
        goarch="arm",
        goarm="6",
        binary_name="gateway-api",
        remote_binary="/usr/local/bin/gateway-api",
        unit_name="gateway-api.service",
        description="Gateway -- serves the UI, sends Wake-on-LAN, and proxies to the compute host",
        environment={
            "COMPUTE_MAC_ADDRESS": secrets.compute_mac_address,
            "GATEWAY_PORT": str(gateway_settings.gateway_port),
            "COMPUTE_HOST": gateway_settings.gateway_proxy_host,
            "COMPUTE_PROXY_PORT": str(gateway_settings.compute_proxy_port),
            "GRAFANA_CLOUD_LOKI_URL": grafana.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": grafana.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": grafana.grafana_cloud_loki_api_key,
        },
    )

    serve_target = f"localhost:{gateway_settings.gateway_port}"
    already_served = serve_target in host.get_fact(TailscaleServeStatus)
    if not already_served:
        server.shell(
            name="Expose the UI/Gateway on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
