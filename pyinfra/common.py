"""Helpers shared by more than one Deploy file.

Not a Deploy file itself -- defines no operations, only fact-derived
helpers and a `device_role` guard -- so it's imported, never passed to
the `pyinfra` CLI directly.
"""

from pyinfra import host
from pyinfra.api import FactBase
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


class TailscaleServeStatus(FactBase):
    """Raw `tailscale serve status --json` output (or `"{}"` if unset, not
    installed, or not yet joined) -- `|| true` keeps that from failing the
    fact gather. The shape isn't officially documented and has changed
    across Tailscale versions -- Tailscale's own docs even note that plain
    `tailscale serve status` and `... --json` can disagree -- so callers
    check for a target string's (e.g. "localhost:5000") presence in this
    raw text rather than parsing a specific key path, which is more
    resilient to that drift than a strict schema parse. Lives here rather
    than in `deploy_tailscale.py` because that file's top level has
    side-effecting operations (installing/joining Tailscale) that would
    re-run if another Deploy file imported from it -- this module is
    side-effect-free by design (see module docstring).
    """

    def command(self) -> str:
        return "tailscale serve status --json 2>/dev/null || true"

    def process(self, output: list[str]) -> str:
        return "\n".join(output).strip() or "{}"
