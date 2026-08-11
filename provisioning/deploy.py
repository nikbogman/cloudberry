"""Single entrypoint composing every Deploy file.

    pyinfra inventory.py deploy.py                 # converge everything
    pyinfra inventory.py deploy.py --dry            # preview everything
    pyinfra inventory.py deploy.py --limit gateway  # converge just gateway's pieces

Each Deploy file is also independently runnable, gated by
`common.has_device_role`. Never invoked automatically -- only run by hand.
"""

from pyinfra import local

# local.include("deploy_tailscale.py")
local.include("deploy_compute_api.py")
local.include("deploy_docker.py")
local.include("deploy_gateway.py")
