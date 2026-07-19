from unittest.mock import MagicMock

import pytest

from control_plane_shared.alloy import AlloyLogger
from control_plane_shared.bind_safety import BindOffTailnetError
from control_server_api.app import create_app

IDENTITY_HEADER = "Tailscale-User-Login"
CONTROL_UI_ORIGIN = "https://control.example.ts.net"
CALLER_IDENTITY = "nicola@example.com"
AUTH_HEADERS = {IDENTITY_HEADER: CALLER_IDENTITY}


@pytest.fixture
def alloy():
    return MagicMock(spec=AlloyLogger)


@pytest.fixture
def app(alloy):
    return create_app(control_ui_origin=CONTROL_UI_ORIGIN, alloy_logger=alloy, bind_host="127.0.0.1")


@pytest.fixture
def client(app):
    return app.test_client()


def test_health_check_requires_identity_header(client):
    response = client.get("/health")

    assert response.status_code == 401


def test_health_check_reports_reachable_with_valid_identity(client):
    response = client.get("/health", headers=AUTH_HEADERS)

    assert response.status_code == 200
    assert response.get_json() == {"reachable": True}


def test_health_check_requires_no_extra_credentials_beyond_identity_header(client):
    # No allow-list of specific identities — tailnet membership alone is sufficient (ADR-0004).
    response = client.get("/health", headers={IDENTITY_HEADER: "anyone-on-the-tailnet@example.com"})

    assert response.status_code == 200


def test_first_health_check_logs_reachability_changed(client, alloy):
    client.get("/health", headers=AUTH_HEADERS)

    alloy.send_event.assert_called_once_with(event_type="reachability_changed", outcome="reachable", identity=None)


def test_subsequent_health_checks_do_not_re_log_reachability(client, alloy):
    client.get("/health", headers=AUTH_HEADERS)
    client.get("/health", headers=AUTH_HEADERS)
    client.get("/health", headers=AUTH_HEADERS)

    assert alloy.send_event.call_count == 1


def test_cors_allows_control_ui_origin(client):
    response = client.get("/health", headers={**AUTH_HEADERS, "Origin": CONTROL_UI_ORIGIN})

    assert response.headers.get("Access-Control-Allow-Origin") == CONTROL_UI_ORIGIN


def test_cors_rejects_other_origins(client):
    response = client.get("/health", headers={**AUTH_HEADERS, "Origin": "https://evil.example.com"})

    assert response.headers.get("Access-Control-Allow-Origin") != "https://evil.example.com"


def test_create_app_refuses_off_tailnet_bind_host(alloy):
    with pytest.raises(BindOffTailnetError):
        create_app(control_ui_origin=CONTROL_UI_ORIGIN, alloy_logger=alloy, bind_host="0.0.0.0")
