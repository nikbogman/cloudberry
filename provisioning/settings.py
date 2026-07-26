"""Typed configuration read from the dev machine's environment at Deploy
time (ADR-0010) -- pydantic-settings validates and coerces types (e.g.
ports as `int`) up front, replacing this repo's previous hand-rolled
`os.environ.get(...)`/`os.environ[...]` calls. Nothing here reads from a
file (no `env_file`) -- ADR-0010 already considered and rejected a
file-based secrets store; these classes are a typed wrapper around plain
environment variables, not a new persistence mechanism.

One settings class per Deploy file's needs. Fields with no default are
required -- pydantic-settings raises a `ValidationError` listing every
missing one at once if they're absent.

Deploy files that only need certain settings when actually targeting a
given device (`has_device_role(...)`) construct the relevant class lazily
inside that guard, exactly where the old `os.environ["X"]` access used to
sit -- so `pyinfra ... deploy_x.py --limit y` still never demands env vars
an unrelated Deploy file/Host group doesn't need, preserving the
"targetable in isolation" contract documented throughout this repo.
"""

from pydantic_settings import BaseSettings


class InventorySettings(BaseSettings):
    # Pure SSH targets -- pyinfra connects to exactly these to run any
    # Deploy file, including deploy_tailscale.py itself (which installs and
    # joins Tailscale in the first place, so this can't assume Tailscale is
    # already reachable). Safe to leave as a plain LAN address permanently
    # -- no separate "bootstrap value" needed -- as long as the
    # gateway/compute devices stay on the same local network as the dev
    # machine, since that keeps working whether or not Tailscale is
    # installed/joined yet.
    gateway_host: str = "pi-zero.tailnet"
    gateway_ssh_user: str = "pi"
    compute_host: str = "main-server.tailnet"
    compute_ssh_user: str = "admin"
    # Each device's real Tailscale MagicDNS name
    # (<device>.<tailnet-name>.ts.net) once joined -- deliberately separate
    # from gateway_host/compute_host above. Those are SSH targets and may
    # well be a bare LAN IP; these two are used only to derive the *other*
    # Deploy file's CORS allow-list / build-time origin
    # (deploy_compute_api.py's UI_ORIGIN, deploy_gateway.py's
    # VITE_COMPUTE_API_URL), which specifically needs the tailnet name
    # since that's the only thing `tailscale serve` issues a valid HTTPS
    # cert for.
    gateway_tailnet_host: str = "pi-zero.your-tailnet-name.ts.net"
    compute_tailnet_host: str = "main-server.your-tailnet-name.ts.net"
    gateway_test_host: str = "localhost"
    gateway_test_ssh_port: int = 2201
    gateway_test_ssh_user: str = "root"
    compute_test_host: str = "localhost"
    compute_test_ssh_port: int = 2202
    compute_test_ssh_user: str = "root"
    # Passed straight through as pyinfra's `_sudo_password` host argument.
    # Needed on compute specifically because its sudo is aliased to sudo-rs
    # (Ubuntu 26.04 default), which prints "sudo: interactive authentication
    # is required" instead of "sudo-rs: ..." -- pyinfra's --use-sudo-password
    # only recognizes the latter, so it never detects the prompt and fails
    # outright rather than asking. Supplying the password up front bypasses
    # that detection entirely (pyinfra uses sudo -A/askpass immediately).
    compute_sudo_password: str | None = None


class TailscaleSettings(BaseSettings):
    tailscale_auth_key: str


class GatewaySettings(BaseSettings):
    # Deliberately not named gateway_host/compute_host: neither class here
    # sets an env_prefix, so those names would silently alias
    # InventorySettings.gateway_host/compute_host (pyinfra's own SSH
    # targets, which may be plain LAN addresses) via the shared env var.
    # These two configure the Gateway binary itself at runtime instead:
    # gateway_bind_host must stay loopback/tailnet-only
    # (AssertTailnetOnlyBind rejects anything else), and gateway_proxy_host
    # is the host it reverse-proxies /server* to, which must be reachable
    # from the Gateway device over the tailnet.
    gateway_bind_host: str = "127.0.0.1"
    gateway_port: int = 5000
    gateway_proxy_host: str = "main-server.tailnet"
    # No default (ADR-0011): the server-side proxy this points at doesn't
    # exist yet, so there's no real value to fall back to.
    compute_proxy_port: int


class GatewaySecrets(BaseSettings):
    compute_mac_address: str
    grafana_cloud_loki_url: str
    grafana_cloud_loki_user: str
    grafana_cloud_loki_api_key: str


class ComputeApiSettings(BaseSettings):
    compute_api_host: str = "127.0.0.1"
    compute_api_port: int = 5000


class ComputeApiSecrets(BaseSettings):
    grafana_cloud_loki_url: str
    grafana_cloud_loki_user: str
    grafana_cloud_loki_api_key: str
