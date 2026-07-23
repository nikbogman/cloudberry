"""Typed configuration read from the dev machine's environment at Deploy
time (ADR-0010) -- pydantic-settings validates and coerces types (e.g.
ports as `int`) up front, replacing this repo's previous hand-rolled
`os.environ.get(...)`/`os.environ[...]` calls. Nothing here reads from a
file (no `env_file`) -- ADR-0010 already considered and rejected a
file-based secrets store; these classes are a typed wrapper around plain
environment variables, not a new persistence mechanism.

One settings class per Deploy file's needs, plus `DeploySourceSettings`
shared by both the Pi API and Server API Deploy files (they pull the same
repo via the same `api_deploy.git_systemd_service` helper). Fields with no
default are required -- pydantic-settings raises a `ValidationError`
listing every missing one at once if they're absent.

Deploy files that only need certain settings when actually targeting a
given device (`has_device_role(...)`) construct the relevant class lazily
inside that guard, exactly where the old `os.environ["X"]` access used to
sit -- so `pyinfra ... deploy_x.py --limit y` still never demands env vars
an unrelated Deploy file/Host group doesn't need, preserving the
"targetable in isolation" contract documented throughout this repo.
"""

from pydantic_settings import BaseSettings


class InventorySettings(BaseSettings):
    pi_host: str = "pi-zero.tailnet"
    pi_ssh_user: str = "pi"
    server_host: str = "main-server.tailnet"
    server_ssh_user: str = "admin"
    pi_test_host: str = "localhost"
    pi_test_ssh_port: int = 2201
    pi_test_ssh_user: str = "root"
    server_test_host: str = "localhost"
    server_test_ssh_port: int = 2202
    server_test_ssh_user: str = "root"


class TailscaleSettings(BaseSettings):
    tailscale_auth_key: str


class DeploySourceSettings(BaseSettings):
    """Where both the Pi API and Server API Deploy files pull this repo from."""

    homelab_repo_url: str = "git@github.com:nikbogman/homelab.git"
    deploy_ref: str = "main"


class PiApiSettings(BaseSettings):
    pi_api_host: str = "127.0.0.1"
    pi_api_port: int = 5000


class PiApiSecrets(BaseSettings):
    server_mac_address: str
    alloy_push_url: str
    server_api_origin: str


class ServerApiSettings(BaseSettings):
    server_api_host: str = "127.0.0.1"
    server_api_port: int = 5000


class ServerApiSecrets(BaseSettings):
    control_ui_origin: str
    alloy_push_url: str


class CaddySettings(BaseSettings):
    main_server_host: str = "main-server.tailnet"
    control_ui_port: int = 8080
    pi_api_port: int = 5000
    # No default (ADR-0011): the server-side proxy this points at doesn't
    # exist yet, so there's no real value to fall back to.
    server_proxy_port: int
