"""Real entrypoint: wires `create_app` to environment-provided config so
this can actually be run (by `flask run`, or a WSGI server under systemd
per ADR-0007), not just imported for tests.
"""

import os

from control_plane_shared.alloy import AlloyLogger

from control_server_api.app import create_app

app = create_app(
    control_ui_origin=os.environ["CONTROL_UI_ORIGIN"],
    alloy_logger=AlloyLogger(push_url=os.environ["ALLOY_PUSH_URL"]),
    bind_host=os.environ.get("CONTROL_SERVER_API_HOST", "127.0.0.1"),
)
