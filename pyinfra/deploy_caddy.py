"""Templates the complete Caddyfile on `pi`.

Single declarative source of truth for every route Caddy serves on the Pi:
the auto-wake-proxy route for each workload service, plus the Control UI's
static files and Control Pi API path-routing. Nothing is hand-edited on the
device -- adding a workload means adding an entry to `WORKLOADS` and
redeploying. Targetable in isolation:

    pyinfra inventory.py deploy_caddy.py --limit pi
    pyinfra inventory.py deploy_caddy.py --limit pi --dry

Requires the tailnet already joined (tailnet-only addresses) and the
Control UI's static path already delivered. Doesn't route to the Control
server API -- that would be a Pi-side relay, which ADR-0001 rejects.

Doesn't build/install the Caddy binary itself (with the `caddy-wol` plugin)
-- only the config. Provisioning the binary is a manual prerequisite for now.
"""

import os

from pyinfra.operations import files, systemd

from common import has_device_kind

WORKLOADS = [
    {"name": "immich", "route_port": os.environ.get("IMMICH_ROUTE_PORT", "8443"), "upstream_port": 2283},
    {"name": "ai-agents", "route_port": os.environ.get("AGENTS_ROUTE_PORT", "8444"), "upstream_port": 8080},
]

if has_device_kind("pi"):
    caddyfile = files.template(
        name="Template the complete Caddyfile",
        src="templates/Caddyfile.j2",
        dest="/etc/caddy/Caddyfile",
        workloads=WORKLOADS,
        main_server_host=os.environ.get("MAIN_SERVER_HOST", "main-server.tailnet"),
        control_ui_root="/srv/control-ui",
        control_ui_port=os.environ.get("CONTROL_UI_PORT", "8080"),
        control_pi_api_port=os.environ.get("CONTROL_PI_API_PORT", "5000"),
    )

    systemd.service(
        name="Reload caddy",
        service="caddy.service",
        running=True,
        enabled=True,
        reloaded=caddyfile.will_change,
    )
