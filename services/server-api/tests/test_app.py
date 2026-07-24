import subprocess
from unittest.mock import MagicMock

import pytest

from control_plane_shared.grafana_cloud import GrafanaCloudLogger
from control_plane_shared.bind_safety import BindOffTailnetError
from server_api.app import create_app
from server_api.suspend import SystemSuspender

IDENTITY_HEADER = "Tailscale-User-Login"
UI_ORIGIN = "https://control.example.ts.net"
CALLER_IDENTITY = "nicola@example.com"
AUTH_HEADERS = {IDENTITY_HEADER: CALLER_IDENTITY}


@pytest.fixture
def event_logger():
    return MagicMock(spec=GrafanaCloudLogger)


@pytest.fixture
def system_suspender():
    return MagicMock(spec=SystemSuspender)


@pytest.fixture
def app(event_logger, system_suspender):
    return create_app(
        ui_origin=UI_ORIGIN,
        event_logger=event_logger,
        system_suspender=system_suspender,
        bind_host="127.0.0.1",
    )


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
    # No allow-list of specific identities — tailnet membership alone is sufficient.
    response = client.get("/health", headers={IDENTITY_HEADER: "anyone-on-the-tailnet@example.com"})

    assert response.status_code == 200


def test_first_health_check_logs_reachability_changed(client, event_logger):
    client.get("/health", headers=AUTH_HEADERS)

    event_logger.send_event.assert_called_once_with(
        event_type="reachability_changed", outcome="reachable", identity=None
    )


def test_subsequent_health_checks_do_not_re_log_reachability(client, event_logger):
    client.get("/health", headers=AUTH_HEADERS)
    client.get("/health", headers=AUTH_HEADERS)
    client.get("/health", headers=AUTH_HEADERS)

    assert event_logger.send_event.call_count == 1


def test_cors_allows_ui_origin(client):
    response = client.get("/health", headers={**AUTH_HEADERS, "Origin": UI_ORIGIN})

    assert response.headers.get("Access-Control-Allow-Origin") == UI_ORIGIN


def test_cors_rejects_other_origins(client):
    response = client.get("/health", headers={**AUTH_HEADERS, "Origin": "https://evil.example.com"})

    assert response.headers.get("Access-Control-Allow-Origin") != "https://evil.example.com"


def test_create_app_refuses_off_tailnet_bind_host(event_logger, system_suspender):
    with pytest.raises(BindOffTailnetError):
        create_app(
            ui_origin=UI_ORIGIN,
            event_logger=event_logger,
            system_suspender=system_suspender,
            bind_host="0.0.0.0",
        )


def test_suspend_requires_identity_header(client):
    response = client.post("/suspend")

    assert response.status_code == 401


def test_suspend_runs_the_configured_system_suspender(client, system_suspender):
    client.post("/suspend", headers=AUTH_HEADERS)

    system_suspender.suspend.assert_called_once_with()


def test_suspend_returns_200_on_success(client):
    response = client.post("/suspend", headers=AUTH_HEADERS)

    assert response.status_code == 200
    assert response.get_json() == {"suspend": "succeeded"}


def test_suspend_logs_requested_then_succeeded_with_caller_identity(client, event_logger):
    client.post("/suspend", headers=AUTH_HEADERS)

    assert event_logger.send_event.call_args_list == [
        ((), {"event_type": "suspend_requested", "outcome": "requested", "identity": CALLER_IDENTITY}),
        ((), {"event_type": "suspend_succeeded", "outcome": "succeeded", "identity": CALLER_IDENTITY}),
    ]


def test_suspend_returns_500_and_logs_failed_when_suspender_raises_called_process_error(
    client, event_logger, system_suspender
):
    system_suspender.suspend.side_effect = subprocess.CalledProcessError(1, ["systemctl", "suspend"])

    response = client.post("/suspend", headers=AUTH_HEADERS)

    assert response.status_code == 500
    assert event_logger.send_event.call_args_list[-1] == (
        (),
        {"event_type": "suspend_failed", "outcome": "failed", "identity": CALLER_IDENTITY},
    )


def test_suspend_returns_500_and_logs_failed_when_suspender_raises_oserror(client, event_logger, system_suspender):
    system_suspender.suspend.side_effect = OSError("command not found")

    response = client.post("/suspend", headers=AUTH_HEADERS)

    assert response.status_code == 500
    assert event_logger.send_event.call_args_list[-1] == (
        (),
        {"event_type": "suspend_failed", "outcome": "failed", "identity": CALLER_IDENTITY},
    )


def test_suspend_does_not_guard_against_repeated_requests(client, system_suspender):
    # The API always attempts the action — the UI's disabled state is the only guard.
    client.post("/suspend", headers=AUTH_HEADERS)
    client.post("/suspend", headers=AUTH_HEADERS)

    assert system_suspender.suspend.call_count == 2
