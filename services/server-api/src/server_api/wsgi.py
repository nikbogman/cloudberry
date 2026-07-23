"""Real entrypoint: wires `create_app` to environment-provided config so
this can actually be run (by `flask run`, or a WSGI server under systemd),
not just imported for tests.
"""

import os

from control_plane_shared.alloy import AlloyLogger

from server_api.app import create_app
from server_api.suspend import SystemSuspender

# No env-driven override of the suspend command — a config knob here could
# let a misconfiguration (e.g. "systemctl poweroff") reintroduce a full
# shutdown path. SystemSuspender's default is the only command this app
# will ever run.
app = create_app(
    control_ui_origin=os.environ["CONTROL_UI_ORIGIN"],
    alloy_logger=AlloyLogger(push_url=os.environ["ALLOY_PUSH_URL"]),
    system_suspender=SystemSuspender(),
    bind_host=os.environ.get("SERVER_API_HOST", "127.0.0.1"),
)
