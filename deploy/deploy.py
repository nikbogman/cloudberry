"""Single entrypoint composing every Deploy file.

    pyinfra inventory.py deploy.py                # converge everything
    pyinfra inventory.py deploy.py --dry          # preview everything
    pyinfra inventory.py deploy.py --limit waker  # converge just waker's pieces

Homelab-wide infrastructure sits at the top level of `deploy/`; each service's
own Deploy files live in a folder named after it. Include paths are relative to
`deploy/` (pyinfra's `state.cwd`), never to this file.

Each Deploy file is also independently runnable, gated by
`common.has_device_role`. Never invoked automatically -- only run by hand.
"""

from pyinfra import local

# local.include("deploy_tailscale.py")
local.include("deploy_docker.py")
local.include("waker-service/sleeper_api.py")
local.include("waker-service/waker.py")
