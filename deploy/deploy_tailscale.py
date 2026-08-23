"""Joins a target device to the tailnet.

Installs Tailscale via the official install script (detects distro and
wires up the apt repo itself), enables `tailscaled`, then runs
`tailscale up` using `TAILSCALE_AUTH_KEY` from the dev machine's
environment -- never written to a file. Targetable in isolation:

    pyinfra inventory.py deploy_tailscale.py --limit pi
    pyinfra inventory.py deploy_tailscale.py --limit server --dry

Install steps also run against `test` stand-ins; the `tailscale up` join
is restricted to `pi`/`server` so a disposable container never enrolls in
the real tailnet.
"""

import json

from pyinfra import host
from pyinfra.api import FactBase
from pyinfra.facts.server import Which
from pyinfra.operations import server, systemd

from common import has_device_role
from settings import TailscaleSettings


class TailscaleBackendState(FactBase):
    """`BackendState` from `tailscale status --json` ("Running" once
    joined), or `None` if not installed/joined. `|| true` keeps a fresh
    host's "command not found" from failing the fact gather.
    """

    def command(self) -> str:
        return "tailscale status --json 2>/dev/null || true"

    def process(self, output: list[str]) -> str | None:
        text = "\n".join(output).strip()
        if not text:
            return None
        try:
            return json.loads(text).get("BackendState")
        except ValueError:
            return None


if has_device_role("pi", "server"):
    if not host.get_fact(Which, command="tailscale"):
        server.shell(
            name="Install Tailscale via the official install script",
            commands=["curl -fsSL https://tailscale.com/install.sh | sh"],
        )

    systemd.service(
        name="Enable and start tailscaled",
        service="tailscaled",
        running=True,
        enabled=True,
    )

    if "pi" in host.groups or "server" in host.groups:
        already_joined = host.get_fact(TailscaleBackendState) == "Running"
        if not already_joined:
            auth_key = TailscaleSettings().tailscale_auth_key
            server.shell(
                name="Join the tailnet",
                commands=[f"tailscale up --authkey={auth_key} --ssh"],
            )
