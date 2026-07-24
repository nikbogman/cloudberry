"""Real entrypoint: wires `create_app` to environment-provided config so
this can actually be run (by `flask run`, or a WSGI server under systemd),
not just imported for tests.
"""

import os

from control_plane_shared.grafana_cloud import GrafanaCloudLogger

from pi_api.app import create_app
from pi_api.wol import WakeOnLanSender

app = create_app(
    server_mac_address=os.environ["SERVER_MAC_ADDRESS"],
    wol_sender=WakeOnLanSender(),
    event_logger=GrafanaCloudLogger(
        loki_url=os.environ["GRAFANA_CLOUD_LOKI_URL"],
        loki_user=os.environ["GRAFANA_CLOUD_LOKI_USER"],
        loki_api_key=os.environ["GRAFANA_CLOUD_LOKI_API_KEY"],
        app="pi-api",
    ),
    bind_host=os.environ.get("PI_API_HOST", "127.0.0.1"),
)
