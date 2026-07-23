"""Tailnet identity-header auth for the control plane's Flask apps.

Both Control APIs sit behind `tailscale serve`, which injects
`Tailscale-User-Login` for any tailnet-authenticated caller. This module
only checks the header is present, not which identity it names.
"""

from functools import wraps

from flask import Response, g, request

IDENTITY_HEADER = "Tailscale-User-Login"


def require_tailnet_identity(view):
    """Reject requests missing the Tailscale identity header with a 401."""

    @wraps(view)
    def wrapped(*args, **kwargs):
        identity = request.headers.get(IDENTITY_HEADER, "").strip()
        if not identity:
            return Response(status=401)
        g.tailnet_identity = identity
        return view(*args, **kwargs)

    return wrapped


def get_caller_identity() -> str:
    """Return the caller's identity captured by `require_tailnet_identity`."""
    return g.tailnet_identity


LOOPBACK_IDENTITY = "auto-wake-proxy"


def require_tailnet_identity_or_loopback(view):
    """Like `require_tailnet_identity`, but also accepts a caller on
    127.0.0.1 with no identity header (ADR-0004, ADR-0012) -- for the
    auto-wake proxy's same-device wake trigger. Opt in per-route; doesn't
    change `require_tailnet_identity` itself.
    """

    @wraps(view)
    def wrapped(*args, **kwargs):
        identity = request.headers.get(IDENTITY_HEADER, "").strip()
        if identity:
            g.tailnet_identity = identity
            return view(*args, **kwargs)
        if request.remote_addr == "127.0.0.1":
            g.tailnet_identity = LOOPBACK_IDENTITY
            return view(*args, **kwargs)
        return Response(status=401)

    return wrapped
