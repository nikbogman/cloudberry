"""Installs and enables the Docker engine on `sleeper`, for workload
Compose stacks (Immich, AI agents, etc.). Targetable in isolation:

    pyinfra inventory.py deploy_docker.py --limit sleeper
    pyinfra inventory.py deploy_docker.py --limit sleeper --dry

Applies to the `test` stand-in too; never to `waker`, which runs no Docker
workloads.

pyinfra has no built-in "install Docker" operation, so this composes
Docker's official apt-repo instructions from apt/systemd primitives.
Distro ID and architecture are read from the host (`common.linux_distro_id`,
`common.DpkgArchitecture`) rather than hardcoded, since `sleeper` isn't a
fixed, known image.
"""

from pyinfra import host
from pyinfra.operations import apt, systemd

from common import DpkgArchitecture, has_device_role, linux_codename, linux_distro_id


if has_device_role("sleeper"):
    codename = linux_codename()
    distro_id = linux_distro_id()
    arch = host.get_fact(DpkgArchitecture)

    apt.key(
        name="Add Docker's apt signing key",
        src=f"https://download.docker.com/linux/{distro_id}/gpg",
        dest="docker.gpg",
    )

    apt.repo(
        name="Add the Docker apt repo",
        src=(
            f"deb [arch={arch} signed-by=/etc/apt/keyrings/docker.gpg] "
            f"https://download.docker.com/linux/{distro_id} {codename} stable"
        ),
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
