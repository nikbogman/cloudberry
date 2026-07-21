"""Ticket 07: templates the complete Caddyfile on `pi`.

The single declarative source of truth (ADR-0008) for every route Caddy
serves on the Pi: the auto-wake-proxy route for each current workload
service, and the Control UI's static files + Control Pi API path-routing.
Nothing is ever hand-edited on the device -- adding a new workload means
adding an entry to `WORKLOADS` below and redeploying. Targetable in
isolation:

    pyinfra inventory.py deploy_caddy.py --limit pi
    pyinfra inventory.py deploy_caddy.py --limit pi --dry

Blocked by (per the ticket): 02 (tailnet must be joined before Caddy binds
tailnet-only addresses) and 05 (needs the Control UI static path this
templates a route for). Does *not* route to the Control server API --
that would be a Pi-side relay, which ADR-0001 explicitly rejects; see
`templates/Caddyfile.j2`'s header comment.

Building/installing the Caddy binary itself (with the `caddy-wol` plugin
compiled in via `xcaddy`, `services/auto_wake_proxy/README.md`'s Build
section) is not this Deploy file's job -- only the config it runs from is
declared here. Provisioning the binary is a documented manual prerequisite
until a future ticket covers it.
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
