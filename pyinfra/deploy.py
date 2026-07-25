"""Single entrypoint composing every Deploy file.

    pyinfra inventory.py deploy.py                 # converge everything
    pyinfra inventory.py deploy.py --dry            # preview everything
    pyinfra inventory.py deploy.py --limit gateway  # converge just gateway's pieces

Each Deploy file is also independently runnable (e.g. `pyinfra inventory.py
deploy_caddy.py --limit gateway`), gated by `common.has_device_role`. This
file just composes them in dependency order: tailnet before Caddy
(tailnet-only addresses), the Gateway/Compute APIs before Caddy (routes to
the UI's static path).

Never invoked automatically -- only run by hand from the dev machine (ADR-0009).
"""

from pyinfra import local

local.include("deploy_tailscale.py")
local.include("deploy_compute_api.py")
local.include("deploy_docker.py")
local.include("deploy_gateway_api.py")
local.include("deploy_caddy.py")
