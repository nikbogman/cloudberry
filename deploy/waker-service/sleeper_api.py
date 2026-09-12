"""Deploys the Sleeper API to `sleeper`.

Cross-compiles the binary on the dev machine and ships it, same as
waker.py -- keeps a Go toolchain off both target devices. Runs
it as an enabled systemd service, then exposes it on the tailnet via
`tailscale serve`. Targetable in isolation:

    pyinfra inventory.py waker-service/sleeper_api.py --limit sleeper
    pyinfra inventory.py waker-service/sleeper_api.py --limit sleeper --dry

Assumes `sleeper` is already joined to the tailnet (`deploy_tailscale.py`,
which `deploy.py` runs first) -- `tailscale serve` needs a joined
`tailscaled`, and this file doesn't re-check that when run standalone.

`UI_ORIGIN` (the CORS allow-list entry) is derived from
`InventorySettings.waker_tailnet_host`, the Waker's real Tailscale
MagicDNS name -- not a separate secret.

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

    # The Sleeper was called Compute until 2026-09-12. Its old unit is
    # still enabled on an already-deployed device and still holds the
    # listen port, so installing the new one isn't enough -- the old one
    # has to go first.
    # ponytail: unconditional shell, delete this block once every device
    # has been deployed at least once since the rename.
    server.shell(
        name="Remove the pre-rename compute-api unit and binaries",
        commands=[
            "systemctl disable --now compute-api.service 2>/dev/null || true",
            "rm -f /etc/systemd/system/compute-api.service"
            " /usr/local/bin/compute-api"
            " /usr/local/bin/compute-api.new"
            " /usr/local/bin/compute-api.swap",
            "systemctl daemon-reload",
        ],
    )

    dpkg_arch = host.get_fact(DpkgArchitecture)
    if dpkg_arch not in GOARCH_BY_DPKG_ARCH:
        raise ValueError(
            f"no known GOARCH for dpkg architecture {dpkg_arch!r} -- "
            f"add it to GOARCH_BY_DPKG_ARCH if this is a real target architecture"
        )

    go_binary_systemd_service(
        module_dir="../waker-service",
        package="./cmd/sleeper-api",
        goos="linux",
        goarch=GOARCH_BY_DPKG_ARCH[dpkg_arch],
        binary_name="sleeper-api",
        remote_binary="/usr/local/bin/sleeper-api",
        unit_name="sleeper-api.service",
        description="Sleeper API -- exposes Suspend and the reachability health check",
        environment={
            "UI_ORIGIN": f"https://{inventory.waker_tailnet_host}",
            "SLEEPER_API_PORT": str(api_settings.sleeper_api_port),
            "GRAFANA_CLOUD_LOKI_URL": grafana.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": grafana.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": grafana.grafana_cloud_loki_api_key,
        },
    )

    serve_target = f"localhost:{api_settings.sleeper_api_port}"
    already_served = serve_target in host.get_fact(TailscaleServeStatus)
    if not already_served:
        server.shell(
            name="Expose the Sleeper API on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
