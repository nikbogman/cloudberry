import importlib

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


def test_wsgi_raises_clearly_when_required_env_vars_are_missing(monkeypatch):
    monkeypatch.delenv("CONTROL_UI_ORIGIN", raising=False)
    monkeypatch.delenv("ALLOY_PUSH_URL", raising=False)

    from control_server_api import wsgi

    with pytest.raises(KeyError):
        importlib.reload(wsgi)
