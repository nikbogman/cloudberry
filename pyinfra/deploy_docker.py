"""Ticket 03: installs and enables the Docker engine on `server`.

So the (out-of-scope, per spec.md) workload Compose stacks -- Immich, AI
agents, etc. -- have an engine to run on. Targetable in isolation:

    pyinfra inventory.py deploy_docker.py --limit server
    pyinfra inventory.py deploy_docker.py --limit server --dry

Applies to any `server`-shaped host, including the `test` stand-in, since
nothing here is destructive to run twice or risky against a disposable
container -- a bad package name surfaces here first (ticket 09). Never
applies to `pi`: the Pi Zero doesn't run Docker workloads (it serves the
Control UI and runs the Control Pi API as a systemd unit, ADR-0007).

pyinfra has no built-in "install Docker engine" operation
(docs/agents/pyinfra.md's Gotchas) -- this composes Docker's own official
apt-repo instructions (https://docs.docker.com/engine/install/debian/)
from pyinfra's generic apt/systemd operations.
"""

from pyinfra.operations import apt, systemd

from common import debian_codename, has_device_kind

if has_device_kind("server"):
    codename = debian_codename()

    apt.key(
        name="Add Docker's apt signing key",
        src="https://download.docker.com/linux/debian/gpg",
        dest="docker.gpg",
    )

    apt.repo(
        name="Add the Docker apt repo",
        src=f"deb [signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian {codename} stable",
        filename="docker",
    )

    apt.packages(
        name="Install the Docker engine",
        packages=[
            "docker-ce",
            "docker-ce-cli",
            "containerd.io",
            "docker-buildx-plugin",
            "docker-compose-plugin",
        ],
        update=True,
    )

    systemd.service(
        name="Enable and start docker",
        service="docker",
        running=True,
        enabled=True,
    )
