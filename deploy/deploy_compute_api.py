"""Deploys the Compute API to `compute`.

Cross-compiles the binary on the dev machine and ships it, same as
deploy_gateway.py -- keeps a Go toolchain off both target devices. Runs
it as an enabled systemd service, then exposes it on the tailnet via
`tailscale serve`. Targetable in isolation:

    pyinfra inventory.py deploy_compute_api.py --limit compute
    pyinfra inventory.py deploy_compute_api.py --limit compute --dry

Assumes `compute` is already joined to the tailnet (`deploy_tailscale.py`,
which `deploy.py` runs first) -- `tailscale serve` needs a joined
`tailscaled`, and this file doesn't re-check that when run standalone.

`UI_ORIGIN` (the CORS allow-list entry) is derived from
`InventorySettings.gateway_tailnet_host`, the Gateway's real Tailscale
MagicDNS name -- not a separate secret.

Required env vars (only when targeting `compute`/its `test` stand-in):
the `GRAFANA_CLOUD_LOKI_*` trio.

`GOARCH` isn't hardcoded, unlike the Pi Zero W's -- `compute` could be
amd64 or arm64, so this reads the real architecture via
`common.DpkgArchitecture`.
"""

from pyinfra import host
from pyinfra.operations import server

from common import DpkgArchitecture, TailscaleServeStatus, has_device_role
from go_build import go_binary_systemd_service
from settings import ComputeApiSettings, GrafanaLokiSecrets, InventorySettings

# dpkg's arch names happen to match Go's GOARCH for both values compute
# realistically runs; anything else fails fast.
GOARCH_BY_DPKG_ARCH = {"amd64": "amd64", "arm64": "arm64"}

api_settings = ComputeApiSettings()
inventory = InventorySettings()

if has_device_role("compute"):
    grafana = GrafanaLokiSecrets()

    dpkg_arch = host.get_fact(DpkgArchitecture)
    if dpkg_arch not in GOARCH_BY_DPKG_ARCH:
        raise ValueError(
            f"no known GOARCH for dpkg architecture {dpkg_arch!r} -- "
            f"add it to GOARCH_BY_DPKG_ARCH if this is a real target architecture"
        )

    go_binary_systemd_service(
        module_dir="../control-plane",
        package="./cmd/compute-api",
        goos="linux",
        goarch=GOARCH_BY_DPKG_ARCH[dpkg_arch],
        binary_name="compute-api",
        remote_binary="/usr/local/bin/compute-api",
        unit_name="compute-api.service",
        description="Compute API -- exposes Suspend and the reachability health check",
        environment={
            "UI_ORIGIN": f"https://{inventory.gateway_tailnet_host}",
            "COMPUTE_API_PORT": str(api_settings.compute_api_port),
            "GRAFANA_CLOUD_LOKI_URL": grafana.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": grafana.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": grafana.grafana_cloud_loki_api_key,
        },
    )

    serve_target = f"localhost:{api_settings.compute_api_port}"
    already_served = serve_target in host.get_fact(TailscaleServeStatus)
    if not already_served:
        server.shell(
            name="Expose the Compute API on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
