"""Installs and enables the Docker engine on `server`.

So workload Compose stacks (Immich, AI agents, etc.) have an engine to run
on. Targetable in isolation:

    pyinfra inventory.py deploy_docker.py --limit server
    pyinfra inventory.py deploy_docker.py --limit server --dry

Applies to the `test` stand-in too -- safe to run twice, so a bad package
name surfaces there first. Never applies to `pi`: the Pi Zero doesn't run
Docker workloads.

pyinfra has no built-in "install Docker engine" operation, so this composes
Docker's official apt-repo instructions
(https://docs.docker.com/engine/install/debian/) from the generic
apt/systemd operations.
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
