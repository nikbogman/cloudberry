"""Deploys the Pi API to `pi`, plus delivers the UI's static build.

Cross-compiles the Pi API's Go binary on the dev machine and ships only
the compiled binary to `pi` -- the Pi Zero W never runs a Go toolchain
(ADR-0013, extended to this app by ADR-0015). Also builds the UI on the
dev machine (`npm ci && npm run build`) and syncs only its static `dist/`
to `pi` -- the Pi never runs a Node/npm toolchain either. Targetable in
isolation:

    pyinfra inventory.py deploy_pi_api.py --limit pi
    pyinfra inventory.py deploy_pi_api.py --limit pi --dry

`SERVER_API_ORIGIN` (baked into the UI build as `VITE_SERVER_API_URL`) is
no longer a separate required secret -- it's derived from
`InventorySettings.server_tailnet_host` (the server's real Tailscale
MagicDNS name, distinct from `server_host`, which is just pyinfra's SSH
target), since that's exactly the origin the Server API's own
`tailscale serve` instance (deploy_server_api.py) exposes it at. The UI,
not this Pi's Caddy, routes to the Server API (ADR-0001 rules out a
Pi-side relay).

Required dev-machine env vars (fail fast if missing, only when targeting
`pi`/its `test` stand-in): `SERVER_MAC_ADDRESS` and the `GRAFANA_CLOUD_LOKI_*`
trio (the Pi API's own config).

Note on `--dry`: the UI build (`local.shell` below) and the Go binary
build (inside `go_binary_systemd_service`) always run, even under `--dry`
-- pyinfra's dry-run guarantee only covers remote operations, not a local
build step. Neither touches `pi`, but isn't a true no-op.
"""

from pyinfra import local
from pyinfra.operations import files

from common import has_device_role
from go_deploy import go_binary_systemd_service
from settings import InventorySettings, PiApiSecrets, PiApiSettings

api_settings = PiApiSettings()
inventory = InventorySettings()

if has_device_role("pi"):
    secrets = PiApiSecrets()

    # Pi Zero W is single-core ARM1176 (armv6) -- GOARM=6. A Pi Zero 2 W
    # would need GOARCH=arm64 (or GOARCH=arm GOARM=7 on a 32-bit OS)
    # instead; update this if the real device ever changes (same note as
    # deploy_caddy.py's wake_plugin build).
    go_binary_systemd_service(
        module_dir="../services",
        package="./cmd/pi-api",
        goos="linux",
        goarch="arm",
        goarm="6",
        binary_name="pi-api",
        remote_binary="/usr/local/bin/pi-api",
        unit_name="pi-api.service",
        description="Pi API -- sends Wake-on-LAN to the main server",
        environment={
            "SERVER_MAC_ADDRESS": secrets.server_mac_address,
            "PI_API_HOST": api_settings.pi_api_host,
            "PI_API_PORT": str(api_settings.pi_api_port),
            "GRAFANA_CLOUD_LOKI_URL": secrets.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": secrets.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": secrets.grafana_cloud_loki_api_key,
        },
    )

    # VITE_SERVER_API_URL bakes in the real origin so the browser calls the
    # Server API directly (ADR-0001). VITE_PI_API_URL is left unset --
    # same-origin works since the Pi API shares this Caddy site.
    local.shell(
        f"cd ../services/ui && npm ci && "
        f"VITE_SERVER_API_URL=https://{inventory.server_tailnet_host} npm run build",
        print_output=True,
    )

    files.sync(
        name="Sync the UI's static build to pi",
        src="../services/ui/dist",
        dest="/srv/ui",
        delete=True,
    )
