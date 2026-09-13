"""Deploys the Waker to `waker`: builds the Waker binary with the
UI embedded in it and ships that one binary. No Go toolchain ever runs on
the Pi Zero W. Targetable in isolation:

    pyinfra inventory.py waker-service/waker.py --limit waker
    pyinfra inventory.py waker-service/waker.py --limit waker --dry

The UI has no build step and no staging copy: `waker-service/ui/ui.go`
`//go:embed`s `waker-service/ui/`'s own contents straight into the binary.
Its one runtime value reaches the browser as `SLEEPER_API_URL`, which the
Waker serves at `/config.js`; it comes from
`InventorySettings.sleeper_tailnet_host` -- the Sleeper API's real
Tailscale MagicDNS name, not a separate secret.

This binary also serves the UI's static files, so it's also the sole thing
standing up `tailscale serve` for this device.

Required env vars (only when targeting `waker`/its `test` stand-in):
`SLEEPER_MAC_ADDRESS` and the `GRAFANA_CLOUD_LOKI_*` trio.

The local Go build always runs, even under `--dry` -- pyinfra's dry-run
guarantee only covers remote operations.
"""

from pyinfra import host
from pyinfra.operations import server

from common import TailscaleServeStatus, has_device_role
from go_build import go_binary_systemd_service
from settings import GrafanaLokiSecrets, InventorySettings, WakerSecrets, WakerSettings

waker_settings = WakerSettings()
inventory = InventorySettings()

if has_device_role("waker"):
    secrets = WakerSecrets()
    grafana = GrafanaLokiSecrets()

    # Pi Zero W is ARM1176 (armv6) -- GOARM=6. A Pi Zero 2 W would need
    # GOARCH=arm64 instead.
    go_binary_systemd_service(
        module_dir="../waker-service",
        package="./cmd/waker-api",
        goos="linux",
        goarch="arm",
        goarm="6",
        binary_name="waker-api",
        remote_binary="/usr/local/bin/waker-api",
        unit_name="waker-api.service",
        description="Waker -- serves the UI and sends Wake-on-LAN",
        environment={
            "SLEEPER_MAC_ADDRESS": secrets.sleeper_mac_address,
            "WAKER_PORT": str(waker_settings.waker_port),
            # Reaches the browser via the Waker's /config.js. No
            # equivalent for the Waker API's own origin: it serves both
            # the UI and /wake, so the UI just uses a relative path.
            "SLEEPER_API_URL": f"https://{inventory.sleeper_tailnet_host}",
            "GRAFANA_CLOUD_LOKI_URL": grafana.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": grafana.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": grafana.grafana_cloud_loki_api_key,
        },
    )

    serve_target = f"localhost:{waker_settings.waker_port}"
    serve_status = host.get_fact(TailscaleServeStatus)
    # Both substrings, not just serve_target: the MagicDNS name is only
    # absent when serve was set up under a since-renamed hostname (as on
    # 2026-09-12's pi0 -> raspberry rename) -- serve_target alone would
    # still match the stale config and skip re-serving under the new name.
    already_served = f"{inventory.waker_tailnet_host}:443" in serve_status and serve_target in serve_status
    if not already_served:
        server.shell(
            name="Expose the UI/Waker on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
