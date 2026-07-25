"""Deploys the Server API to `server`.

Cross-compiles the Server API's Go binary on the dev machine and ships
only the compiled binary to `server` -- built off-device for consistency
with the Pi API (ADR-0013 extended to both apps by ADR-0015), even though
the main server itself isn't resource-constrained; this keeps a Go
toolchain off both target devices rather than just the Pi's. Runs it as
an enabled systemd service, then exposes it on the tailnet over HTTPS via
`tailscale serve` (CONTEXT.md's Server API is "fronted by its own
`tailscale serve` instance" -- this was previously a flagged gap, see
README's Known gaps). Structurally similar to `deploy_pi_api.py` but a
distinct application on a distinct Host group. Targetable in isolation:

    pyinfra inventory.py deploy_server_api.py --limit server
    pyinfra inventory.py deploy_server_api.py --limit server --dry

Assumes `server` is already joined to the tailnet (`deploy_tailscale.py`,
which `deploy.py` always runs first) -- `tailscale serve` needs a running,
joined `tailscaled`, and this file doesn't re-check that when run standalone,
same as every other Deploy file's implicit tailnet-address assumption.

`UI_ORIGIN` (the Server API's CORS allow-list entry) is no longer a
separate required secret -- it's derived from
`InventorySettings.pi_tailnet_host` (the Pi's real Tailscale MagicDNS name,
distinct from `pi_host`, which is just pyinfra's SSH target), since that's
exactly the origin the Pi's own `tailscale serve` instance (deploy_caddy.py)
exposes the UI at.

Required dev-machine env vars (fail fast if missing, only when targeting
`server`/its `test` stand-in): the `GRAFANA_CLOUD_LOKI_*` trio (the Server
API's own config).

`GOARCH` for the off-device build isn't hardcoded -- unlike the Pi Zero W
(a fixed, known device, see `deploy_pi_api.py`), `server` could plausibly
be amd64 or arm64 hardware, so this reads the real architecture from the
host itself via `common.DpkgArchitecture`, the same fact `deploy_docker.py`
already uses for the same reason.
"""

from pyinfra import host
from pyinfra.operations import server

from common import DpkgArchitecture, TailscaleServeStatus, has_device_role
from go_deploy import go_binary_systemd_service
from settings import InventorySettings, ServerApiSecrets, ServerApiSettings

# dpkg's architecture names happen to match Go's GOARCH for both values a
# main server realistically runs; anything else fails fast rather than
# silently guessing.
GOARCH_BY_DPKG_ARCH = {"amd64": "amd64", "arm64": "arm64"}

api_settings = ServerApiSettings()
inventory = InventorySettings()

if has_device_role("server"):
    secrets = ServerApiSecrets()

    dpkg_arch = host.get_fact(DpkgArchitecture)
    if dpkg_arch not in GOARCH_BY_DPKG_ARCH:
        raise ValueError(
            f"no known GOARCH for dpkg architecture {dpkg_arch!r} -- "
            f"add it to GOARCH_BY_DPKG_ARCH if this is a real target architecture"
        )

    go_binary_systemd_service(
        module_dir="../services",
        package="./cmd/server-api",
        goos="linux",
        goarch=GOARCH_BY_DPKG_ARCH[dpkg_arch],
        binary_name="server-api",
        remote_binary="/usr/local/bin/server-api",
        unit_name="server-api.service",
        description="Server API -- exposes Suspend and the reachability health check",
        environment={
            "UI_ORIGIN": f"https://{inventory.pi_tailnet_host}",
            "SERVER_API_HOST": api_settings.server_api_host,
            "SERVER_API_PORT": str(api_settings.server_api_port),
            "GRAFANA_CLOUD_LOKI_URL": secrets.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": secrets.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": secrets.grafana_cloud_loki_api_key,
        },
    )

    serve_target = f"localhost:{api_settings.server_api_port}"
    already_served = serve_target in host.get_fact(TailscaleServeStatus)
    if not already_served:
        server.shell(
            name="Expose the Server API on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
