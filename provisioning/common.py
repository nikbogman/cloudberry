"""Fact-derived helpers and the `device_role` guard shared by Deploy
files. Imported only -- never passed to the pyinfra CLI directly."""

from pyinfra import host
from pyinfra.api import FactBase
from pyinfra.facts.server import LinuxDistribution


class DpkgArchitecture(FactBase):
    """`dpkg --print-architecture` output (e.g. "amd64") -- pyinfra has no
    built-in fact for it, and `server.Arch` wraps `uname -m`'s different
    naming. Used by deploy_docker.py and the Go deploy files to target the
    host's real architecture instead of hardcoding one.
    """

    def command(self) -> str:
        return "dpkg --print-architecture"

    def process(self, output: list[str]) -> str:
        return "\n".join(output).strip()


def linux_distro_id(default: str = "debian") -> str:
    """`ID` from `/etc/os-release` (e.g. "debian", "ubuntu") -- vendor apt
    repos key off this, not the codename. Falls back to `default`.
    """

    release_meta = host.get_fact(LinuxDistribution)["release_meta"]
    return release_meta.get("ID", default)


def linux_codename(default: str = "bookworm") -> str:
    """Debian-family codename (e.g. "bookworm", "noble") for apt source
    lines. Falls back to `default`.
    """

    release_meta = host.get_fact(LinuxDistribution)["release_meta"]
    return release_meta.get("VERSION_CODENAME", default)


def has_device_role(*roles: str) -> bool:
    """Whether the current host has one of the given `device_role`s
    (set in inventory.py). Unlike `host.groups`, true for both a real
    device and its `test` stand-in.
    """

    return host.data.device_role in roles


class TailscaleServeStatus(FactBase):
    """Raw `tailscale serve status --json` output (`"{}"` if unset,
    uninstalled, or unjoined). The JSON shape is undocumented and has
    drifted across Tailscale versions, so callers check for a target
    string (e.g. "localhost:5000") in this raw text rather than parsing a
    key path. Kept here, not in deploy_tailscale.py, because that file's
    top level has side-effecting operations that would re-run on import.
    """

    def command(self) -> str:
        return "tailscale serve status --json 2>/dev/null || true"

    def process(self, output: list[str]) -> str:
        return "\n".join(output).strip() or "{}"
