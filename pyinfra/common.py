"""Helpers shared by more than one Deploy file.

Not a Deploy file itself -- defines no operations, only fact-derived
helpers and a `device_role` guard -- so it's imported, never passed to
the `pyinfra` CLI directly.
"""

from pyinfra import host
from pyinfra.facts.server import LinuxDistribution


def linux_distro_id(default: str = "debian") -> str:
    """`ID` from `/etc/os-release` (e.g. "debian" on Raspberry Pi OS,
    "ubuntu" on Ubuntu Server) -- both are Debian-family, but vendor apt
    repos (Docker, etc.) are hosted under distinct per-distro paths that
    key off this, not the codename. Falls back to `default` if the fact
    can't determine it.
    """

    release_meta = host.get_fact(LinuxDistribution)["release_meta"]
    return release_meta.get("ID", default)


def linux_codename(default: str = "bookworm") -> str:
    """Debian-family codename (e.g. "bookworm" on Raspberry Pi OS, "noble"
    on Ubuntu Server) for apt source lines. Falls back to `default` if the
    fact can't determine it.
    """

    release_meta = host.get_fact(LinuxDistribution)["release_meta"]
    return release_meta.get("VERSION_CODENAME", default)


def has_device_role(*roles: str) -> bool:
    """Whether the current host has one of the given device roles
    ("pi"/"server"), set as `device_role` host data in `inventory.py`.
    Distinct from `host.groups`: true for both a real device and its
    `test` stand-in, so a Deploy file's operations apply to both.
    """

    return host.data.device_role in roles
