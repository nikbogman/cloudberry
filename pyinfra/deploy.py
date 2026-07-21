"""Ticket 08: the single entrypoint composing every Deploy file.

    pyinfra inventory.py deploy.py            # converge everything
    pyinfra inventory.py deploy.py --dry       # preview everything
    pyinfra inventory.py deploy.py --limit pi  # converge just pi's pieces

Each composed Deploy file is independently runnable too (per ticket 02-07,
e.g. `pyinfra inventory.py deploy_caddy.py --limit pi`) -- that's a
property of how they're written (each gated by `common.has_device_kind`), not
something this entrypoint has to implement. This file's only job is
composing them into one command, in dependency order (02 before 07: the
tailnet must be joined before Caddy binds tailnet-only addresses; 05/06
before 07: Caddy routes to the Control UI's static path, which 05
delivers).

Never invoked automatically -- no CI config, cron job, or on-push hook in
this repo calls `deploy.py` (ADR-0009). It only runs when a human runs it
from the dev machine.
"""

from pyinfra import local

local.include("deploy_tailscale.py")
local.include("deploy_docker.py")
local.include("deploy_control_pi_api.py")
local.include("deploy_control_server_api.py")
local.include("deploy_caddy.py")
