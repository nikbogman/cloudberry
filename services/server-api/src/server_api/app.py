"""Server API: exposes a Reachable health check and a Suspend action.

Runs on the main server behind its own `tailscale serve` instance, a
distinct origin from the Control UI — hence the CORS allow-list.
"""

import subprocess

from flask import Flask, Response, jsonify
from flask_cors import CORS

from control_plane_shared.alloy import AlloyLogger
from control_plane_shared.auth import get_caller_identity, require_tailnet_identity
from control_plane_shared.bind_safety import assert_tailnet_only_bind

from server_api.suspend import SystemSuspender


class ReachabilityTracker:
    """Logs a `reachability_changed` event the first time this process
    observes itself as reachable. In-memory state is fine here (the Pi/Server
    APIs are stateless, no DB) — and it's the only "changed" edge this app
    can ever witness: it can't log going *un*reachable, since it's asleep
    while that's true.

    Trade-off: a plain process restart (deploy, crash) emits a fresh event
    even though the server was never actually unreachable — there's no
    persisted state to tell "just woke from suspend" apart from "the Flask
    process restarted while the machine stayed up". Accepted since there's
    no DB/volume to provision or lose; Wake/Suspend events remain the
    precise audit trail regardless.
    """

    def __init__(self) -> None:
        self._has_logged_reachable = False

    def note_reachable(self, alloy: AlloyLogger) -> None:
        if self._has_logged_reachable:
            return
        self._has_logged_reachable = True
        alloy.send_event(event_type="reachability_changed", outcome="reachable", identity=None)


def create_app(
    *, control_ui_origin: str, alloy_logger: AlloyLogger, system_suspender: SystemSuspender, bind_host: str
) -> Flask:
    assert_tailnet_only_bind(bind_host)

    app = Flask(__name__)
    CORS(app, origins=[control_ui_origin])

    reachability = ReachabilityTracker()

    @app.get("/health")
    @require_tailnet_identity
    def health() -> tuple[Response, int]:
        reachability.note_reachable(alloy_logger)
        return jsonify({"reachable": True}), 200

    @app.post("/suspend")
    @require_tailnet_identity
    def suspend() -> tuple[Response, int]:
        identity = get_caller_identity()
        alloy_logger.send_event(event_type="suspend_requested", outcome="requested", identity=identity)

        try:
            system_suspender.suspend()
        except (subprocess.CalledProcessError, OSError):
            alloy_logger.send_event(event_type="suspend_failed", outcome="failed", identity=identity)
            return jsonify({"suspend": "failed"}), 500

        alloy_logger.send_event(event_type="suspend_succeeded", outcome="succeeded", identity=identity)
        return jsonify({"suspend": "succeeded"}), 200

    return app
