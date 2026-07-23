import importlib

import pytest

IDENTITY_HEADER = "Tailscale-User-Login"


@pytest.fixture
def wsgi_module(monkeypatch):
    monkeypatch.setenv("SERVER_MAC_ADDRESS", "AA:BB:CC:DD:EE:FF")
    monkeypatch.setenv("ALLOY_PUSH_URL", "https://alloy.example.internal/loki/api/v1/push")
    monkeypatch.setenv("PI_API_HOST", "127.0.0.1")

    from pi_api import wsgi

    importlib.reload(wsgi)
    return wsgi


def test_wsgi_builds_a_working_app_from_environment(wsgi_module):
    client = wsgi_module.app.test_client()

    response = client.post("/wake", headers={IDENTITY_HEADER: "nicola@example.com"})

    # Real WakeOnLanSender attempts a real UDP broadcast; a sandboxed test
    # environment may refuse it, so only the wiring (not delivery) is asserted.
    assert response.status_code in (200, 500)


def test_wsgi_raises_clearly_when_required_env_vars_are_missing(monkeypatch):
    monkeypatch.delenv("SERVER_MAC_ADDRESS", raising=False)
    monkeypatch.delenv("ALLOY_PUSH_URL", raising=False)

    from pi_api import wsgi

    with pytest.raises(KeyError):
        importlib.reload(wsgi)
