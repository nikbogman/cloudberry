"""Templates the complete Caddyfile on `pi`.

Single declarative source of truth for every route Caddy serves on the Pi:
the UI's static files, Pi API path-routing, and one blind
path-routed proxy to the main server. Nothing is hand-edited on the
device. Targetable in isolation:

    pyinfra inventory.py deploy_caddy.py --limit pi
    pyinfra inventory.py deploy_caddy.py --limit pi --dry

Requires the tailnet already joined (tailnet-only addresses) and the
UI's static path already delivered. Doesn't route to the Server
API -- that would be a Pi-side relay, which ADR-0001 rejects.

Per ADR-0011, this Caddy config carries no knowledge of individual
workload services (Immich, AI agents, etc.) or their container ports --
everything under `/server*` is forwarded, path-stripped, to a single fixed
upstream (`server_proxy_port`) on the main server. Which path reaches
which Docker service is that server's own reverse proxy's job entirely,
out of pyinfra's scope (same boundary as the Compose stacks themselves)
and not yet built -- a known gap until it is (see this repo's README).

Doesn't build/install the Caddy binary itself (with the in-repo
services/pi-proxy/wake_plugin module, ADR-0012) -- only the config.
Provisioning the binary is a manual prerequisite for now.
"""

from pyinfra.operations import files, systemd

from common import has_device_role
from settings import CaddySettings

settings = CaddySettings()

if has_device_role("pi"):
    caddyfile = files.template(
        name="Template the complete Caddyfile",
        src="templates/Caddyfile.j2",
        dest="/etc/caddy/Caddyfile",
        main_server_host=settings.main_server_host,
        server_proxy_port=settings.server_proxy_port,
        ui_root="/srv/ui",
        ui_port=settings.ui_port,
        pi_api_port=settings.pi_api_port,
    )

    systemd.service(
        name="Reload caddy",
        service="caddy.service",
        running=True,
        enabled=True,
        reloaded=caddyfile.will_change,
    )
