"""Tailnet identity-header auth for the control plane's Flask apps.

Both Control APIs sit behind `tailscale serve`, which injects
`Tailscale-User-Login` for any tailnet-authenticated caller. Tailnet
membership is the entire authorization boundary (ADR-0004) — this module
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
