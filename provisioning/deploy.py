"""Single entrypoint composing every Deploy file.

    pyinfra inventory.py deploy.py                 # converge everything
    pyinfra inventory.py deploy.py --dry            # preview everything
    pyinfra inventory.py deploy.py --limit gateway  # converge just gateway's pieces

Each Deploy file is also independently runnable (e.g. `pyinfra inventory.py
deploy_gateway.py --limit gateway`), gated by `common.has_device_role`. This
file just composes them; the only real ordering constraint left is tailnet
before everything else, since the rest rely on tailnet-only addresses.

Never invoked automatically -- only run by hand from the dev machine (ADR-0009).
"""

from pyinfra import local

local.include("deploy_tailscale.py")
local.include("deploy_compute_api.py")
local.include("deploy_docker.py")
local.include("deploy_gateway.py")
