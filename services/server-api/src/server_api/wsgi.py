"""Real entrypoint: wires `create_app` to environment-provided config so
this can actually be run (by `flask run`, or a WSGI server under systemd),
not just imported for tests.
"""

import os

from control_plane_shared.grafana_cloud import GrafanaCloudLogger

from server_api.app import create_app
from server_api.suspend import SystemSuspender

# No env-driven override of the suspend command — a config knob here could
# let a misconfiguration (e.g. "systemctl poweroff") reintroduce a full
# shutdown path. SystemSuspender's default is the only command this app
# will ever run.
app = create_app(
    ui_origin=os.environ["UI_ORIGIN"],
    event_logger=GrafanaCloudLogger(
        loki_url=os.environ["GRAFANA_CLOUD_LOKI_URL"],
        loki_user=os.environ["GRAFANA_CLOUD_LOKI_USER"],
        loki_api_key=os.environ["GRAFANA_CLOUD_LOKI_API_KEY"],
        app="server-api",
    ),
    system_suspender=SystemSuspender(),
    bind_host=os.environ.get("SERVER_API_HOST", "127.0.0.1"),
)
