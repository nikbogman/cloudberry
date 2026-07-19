"""Real entrypoint: wires `create_app` to environment-provided config so
this can actually be run (by `flask run`, or a WSGI server under systemd
per ADR-0007), not just imported for tests.
"""

import os

from control_plane_shared.alloy import AlloyLogger

from control_server_api.app import create_app
from control_server_api.suspend import SystemSuspender

# No env-driven override of the suspend command: ADR-0002 requires that no
# full-shutdown (ACPI S5) code path exists anywhere in this app, and a
# config knob here would let a misconfiguration (e.g. "systemctl poweroff")
# quietly reintroduce one. SystemSuspender's default is the only command
# this app will ever run.
app = create_app(
    control_ui_origin=os.environ["CONTROL_UI_ORIGIN"],
    alloy_logger=AlloyLogger(push_url=os.environ["ALLOY_PUSH_URL"]),
    system_suspender=SystemSuspender(),
    bind_host=os.environ.get("CONTROL_SERVER_API_HOST", "127.0.0.1"),
)
