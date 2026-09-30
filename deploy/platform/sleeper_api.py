"""Deploys the Sleeper API to `sleeper`.

Cross-compiles the binary on the dev machine and ships it, same as waker.py --
keeps a Go toolchain off both target devices. Runs it as an enabled systemd
service, then exposes it on the tailnet via `tailscale serve`. Targetable in
isolation:

    pyinfra inventory.py platform/sleeper_api.py --limit sleeper
    pyinfra inventory.py platform/sleeper_api.py --limit sleeper --dry

Assumes `sleeper` is already joined to the tailnet (`deploy_tailscale.py`,
which `deploy.py` runs first) -- `tailscale serve` needs a joined
`tailscaled`, and this file doesn't re-check that when run standalone.

Required env vars (only when targeting `sleeper`/its `test` stand-in):
the `GRAFANA_CLOUD_LOKI_*` trio.

`GOARCH` isn't hardcoded, unlike the Pi Zero W's -- `sleeper` could be
amd64 or arm64, so this reads the real architecture via
`common.DpkgArchitecture`.
"""

from pyinfra import host
from pyinfra.operations import server

from common import DpkgArchitecture, TailscaleServeStatus, has_device_role
from go_build import go_binary_systemd_service
from settings import GrafanaLokiSecrets, InventorySettings, SleeperApiSettings

# dpkg's arch names happen to match Go's GOARCH for both values sleeper
# realistically runs; anything else fails fast.
GOARCH_BY_DPKG_ARCH = {"amd64": "amd64", "arm64": "arm64"}

api_settings = SleeperApiSettings()
inventory = InventorySettings()

if has_device_role("sleeper"):
    grafana = GrafanaLokiSecrets()

    dpkg_arch = host.get_fact(DpkgArchitecture)
    if dpkg_arch not in GOARCH_BY_DPKG_ARCH:
        raise ValueError(
            f"no known GOARCH for dpkg architecture {dpkg_arch!r} -- "
            f"add it to GOARCH_BY_DPKG_ARCH if this is a real target architecture"
        )

    go_binary_systemd_service(
        module_dir="../platform",
        package="./cmd/sleeper-api",
        goos="linux",
        goarch=GOARCH_BY_DPKG_ARCH[dpkg_arch],
        binary_name="sleeper-api",
        remote_binary="/usr/local/bin/sleeper-api",
        unit_name="sleeper-api.service",
        description="Sleeper API -- exposes Suspend and the reachability health check",
        environment={
            "SLEEPER_API_PORT": str(api_settings.sleeper_api_port),
            "GRAFANA_CLOUD_LOKI_URL": grafana.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": grafana.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": grafana.grafana_cloud_loki_api_key,
        },
    )

    serve_target = f"localhost:{api_settings.sleeper_api_port}"
    serve_status = host.get_fact(TailscaleServeStatus)
    # Both substrings, not just serve_target: see waker.py's matching check.
    already_served = f"{inventory.sleeper_tailnet_host}:443" in serve_status and serve_target in serve_status
    if not already_served:
        server.shell(
            name="Expose the Sleeper API on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
