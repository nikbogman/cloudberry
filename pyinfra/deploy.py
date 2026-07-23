"""Single entrypoint composing every Deploy file.

    pyinfra inventory.py deploy.py            # converge everything
    pyinfra inventory.py deploy.py --dry       # preview everything
    pyinfra inventory.py deploy.py --limit pi  # converge just pi's pieces

Each Deploy file is also independently runnable (e.g. `pyinfra inventory.py
deploy_caddy.py --limit pi`), gated by `common.has_device_role`. This file
just composes them in dependency order: tailnet before Caddy (tailnet-only
addresses), the Pi/Server APIs before Caddy (routes to the UI's
static path).

Never invoked automatically -- only run by hand from the dev machine (ADR-0009).
"""

from pyinfra import local

local.include("deploy_tailscale.py")
local.include("deploy_docker.py")
local.include("deploy_pi_api.py")
local.include("deploy_server_api.py")
local.include("deploy_caddy.py")
