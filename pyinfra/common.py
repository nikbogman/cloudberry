"""Helpers shared by more than one Deploy file.

Not a Deploy file itself -- defines no operations, only fact-derived
helpers and a `device_kind` guard -- so it's imported, never passed to
the `pyinfra` CLI directly.
"""

from pyinfra import host
from pyinfra.facts.server import LinuxDistribution


def debian_codename(default: str = "bookworm") -> str:
    """Debian/Raspberry Pi OS codename (e.g. "bookworm") for apt source
    lines. Falls back to `default` if the fact can't determine it.
    """

    release_meta = host.get_fact(LinuxDistribution)["release_meta"]
    return release_meta.get("VERSION_CODENAME", default)


def has_device_kind(*kinds: str) -> bool:
    """Whether the current host is one of the given device kinds
    ("pi"/"server"), set as `device_kind` host data in `inventory.py`.
    Distinct from `host.groups`: true for both a real device and its
    `test` stand-in, so a Deploy file's operations apply to both.
    """

    return host.data.device_kind in kinds
