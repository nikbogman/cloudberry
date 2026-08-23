"""Typed configuration read from the dev machine's environment at Deploy
time. One settings class per Deploy file's needs; fields with no default
are required, and pydantic-settings raises a `ValidationError` listing
every missing one at once. No `env_file` support -- these are a typed
wrapper around plain environment variables, not a persistence mechanism.

Deploy files construct a class lazily inside its `has_device_role(...)`
guard, so `pyinfra ... deploy_x.py --limit y` never demands env vars an
unrelated Deploy file doesn't need.
"""

from pydantic_settings import BaseSettings


class InventorySettings(BaseSettings):
    # Pure SSH targets -- pyinfra connects to exactly these, including for
    # deploy_tailscale.py itself, so this can't assume a tailnet address.
    # Safe to leave as a plain LAN address permanently.
    gateway_host: str = "pi-zero.tailnet"
    gateway_ssh_user: str = "pi"
    compute_host: str = "main-server.tailnet"
    compute_ssh_user: str = "admin"
    # Real Tailscale MagicDNS names, separate from the SSH targets above
    # (which may be bare LAN IPs). Used to derive the *other* Deploy
    # file's CORS/build-time origin, since only the MagicDNS name gets a
    # valid `tailscale serve` HTTPS cert.
    gateway_tailnet_host: str = "pi-zero.your-tailnet-name.ts.net"
    compute_tailnet_host: str = "main-server.your-tailnet-name.ts.net"
    gateway_test_host: str = "localhost"
    gateway_test_ssh_port: int = 2201
    gateway_test_ssh_user: str = "root"
    compute_test_host: str = "localhost"
    compute_test_ssh_port: int = 2202
    compute_test_ssh_user: str = "root"
    # Passed as pyinfra's `_sudo_password` host arg. Needed on compute
    # because its sudo is aliased to sudo-rs, whose prompt text
    # pyinfra's --use-sudo-password doesn't recognize.
    compute_sudo_password: str | None = None


class TailscaleSettings(BaseSettings):
    tailscale_auth_key: str


class GrafanaLokiSecrets(BaseSettings):
    # Shared by deploy_gateway.py and deploy_compute_api.py -- both
    # binaries log to the same Grafana Cloud Loki endpoint.
    grafana_cloud_loki_url: str
    grafana_cloud_loki_user: str
    grafana_cloud_loki_api_key: str


class GatewaySettings(BaseSettings):
    # Not named gateway_host/compute_host -- no env_prefix here, so those
    # names would alias InventorySettings' SSH targets via the shared env
    # var. These configure the Gateway binary at runtime instead.
    #
    # No bind-host field: the Gateway binary defaults GATEWAY_HOST to
    # 127.0.0.1 itself (control-plane/internal/gateway/config.go), and
    # that's the only value it's ever run with here, so there's nothing
    # for deploy to override.
    gateway_port: int = 5000
    # gateway_proxy_host is what /server* reverse-proxies to -- distinct
    # from InventorySettings.compute_host despite the matching default:
    # that one is pyinfra's SSH target, this is the Gateway binary's own
    # runtime target, and they're allowed to diverge (e.g. a LAN IP vs. a
    # tailnet name) even though today they're the same box.
    gateway_proxy_host: str = "main-server.tailnet"
    # No default: the server-side proxy this points at doesn't exist yet.
    compute_proxy_port: int


class GatewaySecrets(BaseSettings):
    compute_mac_address: str


class ComputeApiSettings(BaseSettings):
    # No bind-host field, matching GatewaySettings above -- the Compute
    # API binary's own COMPUTE_API_HOST default (127.0.0.1) is never
    # overridden here.
    compute_api_port: int = 5000
