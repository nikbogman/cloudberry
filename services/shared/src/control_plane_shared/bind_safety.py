"""Startup assertion that a control-plane app is not reachable off-tailnet.

Allowed: loopback, and Tailscale's own address ranges (the CGNAT IPv4 range
it assigns nodes, and its IPv6 ULA range).
"""

import ipaddress
import socket

TAILSCALE_IPV4_RANGE = ipaddress.ip_network("100.64.0.0/10")
TAILSCALE_IPV6_RANGE = ipaddress.ip_network("fd7a:115c:a1e0::/48")


class BindOffTailnetError(ValueError):
    """Raised when a host would bind the app somewhere reachable off-tailnet."""


def assert_tailnet_only_bind(host: str) -> None:
    """Raise `BindOffTailnetError` unless `host` is loopback or a tailnet address."""
    if host == "localhost":
        return

    try:
        address = ipaddress.ip_address(host)
    except ValueError:
        try:
            address = ipaddress.ip_address(socket.gethostbyname(host))
        except (socket.gaierror, ValueError) as exc:
            raise BindOffTailnetError(f"could not resolve host {host!r} to check its bind safety") from exc

    if address.is_loopback:
        return
    if address in TAILSCALE_IPV4_RANGE or address in TAILSCALE_IPV6_RANGE:
        return

    raise BindOffTailnetError(
        f"refusing to bind to {host!r}: not loopback or a tailnet address, "
        "which would expose this control-plane app off-tailnet"
    )
