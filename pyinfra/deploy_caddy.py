"""Builds the wake_plugin-enabled Caddy binary and templates the complete
Caddyfile on `pi`.

Single declarative source of truth for every route Caddy serves on the Pi:
the UI's static files, Pi API path-routing, and one blind
path-routed proxy to the main server. Nothing is hand-edited on the
device. Targetable in isolation:

    pyinfra inventory.py deploy_caddy.py --limit pi
    pyinfra inventory.py deploy_caddy.py --limit pi --dry

Requires the tailnet already joined (tailnet-only addresses) and the
UI's static path already delivered. Doesn't route to the Server
API -- that would be a Pi-side relay, which ADR-0001 rejects.

After Caddy is up, also runs `tailscale serve --bg --https=443` against
Caddy's `ui_port` so the UI/Pi API are actually reachable on the tailnet
over HTTPS (CONTEXT.md's UI: "served by Caddy on the Pi Zero via
`tailscale serve`") -- this was the other half of a previously flagged gap,
see README's Known gaps; `deploy_server_api.py` closed the Server API half.
Guarded the same way, via `common.TailscaleServeStatus`, so a second run
is a no-op.

Per ADR-0011, this Caddy config carries no knowledge of individual
workload services (Immich, AI agents, etc.) or their container ports --
everything under `/server*` is forwarded, path-stripped, to a single fixed
upstream (`server_proxy_port`) on the main server. Which path reaches
which Docker service is that server's own reverse proxy's job entirely,
out of pyinfra's scope (same boundary as the Compose stacks themselves)
and not yet built -- a known gap until it is (see this repo's README).

Builds the Caddy binary (with the in-repo services/pi-proxy/wake_plugin
module, ADR-0012) on the dev machine via `xcaddy`, cross-compiled for the
Pi Zero W, and ships only the compiled binary -- the same off-device
build-then-ship shape ADR-0006 already uses for the UI's Vite build, and
for the same reason: the Pi Zero W never runs a Go toolchain any more than
it runs Node (ADR-0013). Requires `go` and `xcaddy`
(https://github.com/caddyserver/xcaddy) installed on the dev machine.

Note on `--dry`: the local xcaddy build always runs, even under `--dry` --
pyinfra's dry-run guarantee only covers remote operations, not a local
build step (same caveat deploy_pi_api.py's UI build documents).
"""

from pyinfra import host, local
from pyinfra.operations import files, server, systemd

from common import TailscaleServeStatus, has_device_role
from settings import CaddySettings

settings = CaddySettings()

WAKE_PLUGIN_MODULE = "github.com/nikbogman/homelab/services/pi-proxy/wake_plugin"
WAKE_PLUGIN_DIR = "../services/pi-proxy/wake_plugin"
LOCAL_BINARY = f"{WAKE_PLUGIN_DIR}/dist/caddy"
REMOTE_BINARY = "/usr/local/bin/caddy"

if has_device_role("pi"):
    # Pi Zero W is single-core ARM1176 (armv6) -- GOARM=6. A Pi Zero 2 W
    # would need GOARCH=arm64 (or GOARCH=arm GOARM=7 on a 32-bit OS)
    # instead; update this constant if the real device ever changes.
    local.shell(
        f"mkdir -p {WAKE_PLUGIN_DIR}/dist && "
        f"GOOS=linux GOARCH=arm GOARM=6 xcaddy build "
        f"--with {WAKE_PLUGIN_MODULE}={WAKE_PLUGIN_DIR} "
        f"--output {LOCAL_BINARY}",
        print_output=True,
    )

    files.directory(
        name="Create /etc/caddy",
        path="/etc/caddy",
    )

    binary = files.put(
        name="Ship the wake_plugin-enabled caddy binary to pi",
        src=LOCAL_BINARY,
        dest=REMOTE_BINARY,
        mode="755",
    )

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

    unit = files.template(
        name="Install the caddy systemd unit",
        src="templates/caddy.service.j2",
        dest="/etc/systemd/system/caddy.service",
        binary_path=REMOTE_BINARY,
        run_as_user="root",
    )

    # A new/changed binary or unit needs a real restart (a reload can't pick
    # up either); a Caddyfile-only change stays reload-eligible, matching
    # ADR-0008's prior reload-only behavior for config changes.
    needs_restart = binary.will_change or unit.will_change

    systemd.service(
        name="Enable/restart/reload caddy",
        service="caddy.service",
        running=True,
        enabled=True,
        restarted=needs_restart,
        reloaded=caddyfile.will_change and not needs_restart,
        daemon_reload=unit.will_change,
    )

    serve_target = f"localhost:{settings.ui_port}"
    already_served = serve_target in host.get_fact(TailscaleServeStatus)
    if not already_served:
        server.shell(
            name="Expose the UI/Pi API on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
