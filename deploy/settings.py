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
    waker_host: str = "pi-zero.tailnet"
    waker_ssh_user: str = "pi"
    sleeper_host: str = "main-server.tailnet"
    sleeper_ssh_user: str = "admin"
    # Real Tailscale MagicDNS names, separate from the SSH targets above
    # (which may be bare LAN IPs). Used to check each device's
    # `tailscale serve` config, which is keyed by the MagicDNS name.
    waker_tailnet_host: str = "pi-zero.your-tailnet-name.ts.net"
    sleeper_tailnet_host: str = "main-server.your-tailnet-name.ts.net"
    waker_test_host: str = "localhost"
    waker_test_ssh_port: int = 2201
    waker_test_ssh_user: str = "root"
    sleeper_test_host: str = "localhost"
    sleeper_test_ssh_port: int = 2202
    sleeper_test_ssh_user: str = "root"
    # Passed as pyinfra's `_sudo_password` host arg. Needed on sleeper
    # because its sudo is aliased to sudo-rs, whose prompt text
    # pyinfra's --use-sudo-password doesn't recognize.
    sleeper_sudo_password: str | None = None


class TailscaleSettings(BaseSettings):
    tailscale_auth_key: str


class GrafanaLokiSecrets(BaseSettings):
    # Shared by platform/waker.py and platform/sleeper_api.py --
    # both
    # binaries log to the same Grafana Cloud Loki endpoint.
    grafana_cloud_loki_url: str
    grafana_cloud_loki_user: str
    grafana_cloud_loki_api_key: str


class WakerSettings(BaseSettings):
    # Not named waker_host -- no env_prefix here, so that name would
    # alias InventorySettings' SSH target via the shared env var. This
    # configures the Waker binary at runtime instead.
    waker_port: int = 5000


class WakerSecrets(BaseSettings):
    sleeper_mac_address: str


class SleeperApiSettings(BaseSettings):
    sleeper_api_port: int = 5000
