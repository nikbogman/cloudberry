"""Ticket 02: joins a target device to the tailnet.

Installs Tailscale from its official apt repo, enables `tailscaled`, then
runs `tailscale up` with an auth key read from the `TAILSCALE_AUTH_KEY`
environment variable on the dev machine (ADR-0010) -- the key is never
written to a file in this repo. Targetable in isolation:

    pyinfra inventory.py deploy_tailscale.py --limit pi
    pyinfra inventory.py deploy_tailscale.py --limit server --dry

The install steps run for any `pi`/`server`-shaped host, including `test`
stand-ins, so a bad package name or repo config error surfaces there first
(ticket 09). The actual `tailscale up` join is restricted to the real
`pi`/`server` Host groups -- never `test` -- so a disposable-container run
never enrolls an ephemeral container into the real tailnet with the real
auth key.
"""

import json
import os

from pyinfra import host
from pyinfra.api import FactBase
from pyinfra.operations import apt, server, systemd

from common import debian_codename, has_device_kind


class TailscaleBackendState(FactBase):
    """`BackendState` from `tailscale status --json` (e.g. "Running" once
    joined), or `None` if tailscale isn't installed/joined yet. Read during
    prepare, before this file's own install operations have executed --
    the `|| true` keeps a "command not found" on a fresh host from failing
    the fact gather; it's simply treated as "not joined".
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


if has_device_kind("pi", "server"):
    codename = debian_codename()

    apt.key(
        name="Add Tailscale's apt signing key",
        src=f"https://pkgs.tailscale.com/stable/debian/{codename}.noarmor.gpg",
        dest="tailscale.gpg",
    )

    apt.repo(
        name="Add the Tailscale apt repo",
        src=f"deb [signed-by=/etc/apt/keyrings/tailscale.gpg] https://pkgs.tailscale.com/stable/debian {codename} main",
        filename="tailscale",
    )

    apt.packages(
        name="Install tailscale",
        packages=["tailscale"],
        update=True,
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
            auth_key = os.environ["TAILSCALE_AUTH_KEY"]
            server.shell(
                name="Join the tailnet",
                commands=[f"tailscale up --authkey={auth_key} --ssh"],
            )
