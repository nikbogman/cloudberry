"""Real entrypoint: wires `create_app` to environment-provided config so
this can actually be run (by `flask run`, or a WSGI server under systemd
per ADR-0007), not just imported for tests.
"""

import os

from control_plane_shared.alloy import AlloyLogger

from control_pi_api.app import create_app
from control_pi_api.wol import WakeOnLanSender

app = create_app(
    server_mac_address=os.environ["SERVER_MAC_ADDRESS"],
    wol_sender=WakeOnLanSender(),
    alloy_logger=AlloyLogger(push_url=os.environ["ALLOY_PUSH_URL"]),
    bind_host=os.environ.get("CONTROL_PI_API_HOST", "127.0.0.1"),
)
