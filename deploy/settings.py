"""Typed env-var config read from the dev machine at Deploy time, one class
per Deploy file's needs. Variables and defaults: docs/deploy.md#configuration.

No `env_file` support -- a typed wrapper around plain environment variables,
not a persistence mechanism.
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
    # Not named gateway_host -- no env_prefix here, so that name would
    # alias InventorySettings' SSH target via the shared env var. This
    # configures the Gateway binary at runtime instead.
    gateway_port: int = 5000


class GatewaySecrets(BaseSettings):
    compute_mac_address: str


class ComputeApiSettings(BaseSettings):
    compute_api_port: int = 5000
