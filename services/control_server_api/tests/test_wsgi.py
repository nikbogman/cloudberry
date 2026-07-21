import importlib
import subprocess
from unittest.mock import MagicMock

import pytest

IDENTITY_HEADER = "Tailscale-User-Login"


@pytest.fixture
def wsgi_module(monkeypatch):
    monkeypatch.setenv("CONTROL_UI_ORIGIN", "https://control.example.ts.net")
    monkeypatch.setenv("ALLOY_PUSH_URL", "https://alloy.example.internal/loki/api/v1/push")
    monkeypatch.setenv("CONTROL_SERVER_API_HOST", "127.0.0.1")

    from control_server_api import wsgi

    importlib.reload(wsgi)
    return wsgi


def test_wsgi_builds_a_working_app_from_environment(wsgi_module):
    client = wsgi_module.app.test_client()

    response = client.get("/health", headers={IDENTITY_HEADER: "nicola@example.com"})

    assert response.status_code == 200
    assert response.get_json() == {"reachable": True}


def test_wsgi_wires_suspend_to_the_real_systemctl_suspend_command(wsgi_module, monkeypatch):
    # A real "systemctl suspend" must never run in a test — patch the one
    # real side effect so this still exercises the actual subprocess.run
    # call the wiring makes.
    run_mock = MagicMock()
    monkeypatch.setattr(subprocess, "run", run_mock)

    client = wsgi_module.app.test_client()
    response = client.post("/suspend", headers={IDENTITY_HEADER: "nicola@example.com"})

    assert response.status_code == 200
    assert response.get_json() == {"suspend": "succeeded"}
    run_mock.assert_called_once_with(("systemctl", "suspend"), check=True)


def test_wsgi_raises_clearly_when_required_env_vars_are_missing(monkeypatch):
    monkeypatch.delenv("CONTROL_UI_ORIGIN", raising=False)
    monkeypatch.delenv("ALLOY_PUSH_URL", raising=False)

    from control_server_api import wsgi

    with pytest.raises(KeyError):
        importlib.reload(wsgi)
