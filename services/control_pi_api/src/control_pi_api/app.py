"""Control Pi API: sends a Wake-on-LAN packet to the main server on request.

Runs on the Pi Zero, path-routed under the same `tailscale serve` app as
the Control UI — same-origin, so no CORS entry is needed (unlike the
Control server API, a distinct origin per ADR-0001).
"""

from flask import Flask, Response, jsonify

from control_plane_shared.alloy import AlloyLogger
from control_plane_shared.auth import get_caller_identity, require_tailnet_identity
from control_plane_shared.bind_safety import assert_tailnet_only_bind

from control_pi_api.wol import WakeOnLanSender, build_magic_packet


def create_app(*, server_mac_address: str, wol_sender: WakeOnLanSender, alloy_logger: AlloyLogger, bind_host: str) -> Flask:
    assert_tailnet_only_bind(bind_host)
    build_magic_packet(server_mac_address)  # fail fast on a malformed MAC at startup, not mid-request

    app = Flask(__name__)

    @app.post("/wake")
    @require_tailnet_identity
    def wake() -> tuple[Response, int]:
        identity = get_caller_identity()
        alloy_logger.send_event(event_type="wake_requested", outcome="requested", identity=identity)

        try:
            wol_sender.send(server_mac_address)
        except OSError:
            alloy_logger.send_event(event_type="wake_failed", outcome="failed", identity=identity)
            return jsonify({"wake": "failed"}), 500

        alloy_logger.send_event(event_type="wake_succeeded", outcome="succeeded", identity=identity)
        return jsonify({"wake": "succeeded"}), 200

    return app
