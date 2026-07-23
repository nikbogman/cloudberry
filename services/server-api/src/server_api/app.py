"""Server API: exposes a Reachable health check and a Suspend action.

Runs on the main server behind its own `tailscale serve` instance, a
distinct origin from the UI — hence the CORS allow-list.
"""

import subprocess

from flask import Flask, Response, jsonify
from flask_cors import CORS

from control_plane_shared.auth import require_tailnet_identity
from control_plane_shared.bind_safety import assert_tailnet_only_bind

from server_api.suspend import SystemSuspender


def create_app(*, ui_origin: str, system_suspender: SystemSuspender, bind_host: str) -> Flask:
    assert_tailnet_only_bind(bind_host)

    app = Flask(__name__)
    CORS(app, origins=[ui_origin])

    @app.get("/health")
    @require_tailnet_identity
    def health() -> tuple[Response, int]:
        return jsonify({"reachable": True}), 200

    @app.post("/suspend")
    @require_tailnet_identity
    def suspend() -> tuple[Response, int]:
        try:
            system_suspender.suspend()
        except (subprocess.CalledProcessError, OSError):
            return jsonify({"suspend": "failed"}), 500

        return jsonify({"suspend": "succeeded"}), 200

    return app
