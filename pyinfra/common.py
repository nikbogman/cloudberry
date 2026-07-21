"""Helpers shared by more than one Deploy file.

Not a Deploy file itself -- defines no operations, only fact-derived
helpers and a `device_kind` guard -- so it's imported, never passed to
the `pyinfra` CLI directly.
"""

from pyinfra import host
from pyinfra.facts.server import LinuxDistribution


def debian_codename(default: str = "bookworm") -> str:
    """The target's Debian/Raspberry Pi OS codename (e.g. "bookworm"), for
    apt source lines. Falls back to `default` when the fact can't determine
    it -- `pi` and `server` are both assumed Debian-family per this repo's
    provisioning scope (`.scratch/pyinfra-provisioning/spec.md`).
    """

    release_meta = host.get_fact(LinuxDistribution)["release_meta"]
    return release_meta.get("VERSION_CODENAME", default)


def has_device_kind(*kinds: str) -> bool:
    """Whether the current host is one of the given device kinds
    ("pi"/"server"), set as `device_kind` host data in `inventory.py`.
    Deliberately distinct from `host.groups` (CONTEXT.md's Host group
    entry warns against overloading "role" in the Ansible sense for that
    concept): it's true for a real device *and* its disposable `test`
    stand-in alike, so a Deploy file's operations apply to both without
    also matching an unrelated host of the other kind.
    """

    return host.data.device_kind in kinds
