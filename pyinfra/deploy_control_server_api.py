"""Deploys the Control server API to `server`.

Uses the shared helper to pull this repo to a specific ref and run the
Control server API as an enabled systemd service. Structurally similar to
`deploy_control_pi_api.py` but a distinct application on a distinct Host
group. Targetable in isolation:

    pyinfra inventory.py deploy_control_server_api.py --limit server
    pyinfra inventory.py deploy_control_server_api.py --limit server --dry

Required dev-machine env vars (fail fast if missing, only when targeting
`server`/its `test` stand-in): `CONTROL_UI_ORIGIN`, `ALLOY_PUSH_URL` (the
Control server API's own config).
"""

import os

from common import has_device_kind
from control_api_deploy import git_systemd_service

REPO_URL = os.environ.get("HOMELAB_REPO_URL", "git@github.com:nikbogman/homelab.git")
REF = os.environ.get("DEPLOY_REF", "main")
CLONE_DEST = "/srv/homelab"
APP_DIR = f"{CLONE_DEST}/services/control_server_api"
BIND_HOST = os.environ.get("CONTROL_SERVER_API_HOST", "127.0.0.1")
BIND_PORT = os.environ.get("CONTROL_SERVER_API_PORT", "5000")

if has_device_kind("server"):
    git_systemd_service(
        repo_url=REPO_URL,
        ref=REF,
        dest=CLONE_DEST,
        working_directory=APP_DIR,
        unit_name="control-server-api.service",
        description="Control server API -- exposes Suspend and the reachability health check",
        start_command=f"uv run flask --app control_server_api.wsgi run --host {BIND_HOST} --port {BIND_PORT}",
        environment={
            "CONTROL_UI_ORIGIN": os.environ["CONTROL_UI_ORIGIN"],
            "ALLOY_PUSH_URL": os.environ["ALLOY_PUSH_URL"],
            "CONTROL_SERVER_API_HOST": BIND_HOST,
        },
        setup_commands=["uv sync"],
    )
