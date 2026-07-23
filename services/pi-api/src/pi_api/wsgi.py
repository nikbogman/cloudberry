"""Real entrypoint: wires `create_app` to environment-provided config so
this can actually be run (by `flask run`, or a WSGI server under systemd),
not just imported for tests.
"""

import os

from pi_api.app import create_app
from pi_api.wol import WakeOnLanSender

app = create_app(
    server_mac_address=os.environ["SERVER_MAC_ADDRESS"],
    wol_sender=WakeOnLanSender(),
    bind_host=os.environ.get("PI_API_HOST", "127.0.0.1"),
)
