from unittest.mock import MagicMock

import pytest

from pi_api.app import create_app
from pi_api.wol import WakeOnLanSender
from control_plane_shared.alloy import AlloyLogger
from control_plane_shared.bind_safety import BindOffTailnetError

IDENTITY_HEADER = "Tailscale-User-Login"
CALLER_IDENTITY = "nicola@example.com"
AUTH_HEADERS = {IDENTITY_HEADER: CALLER_IDENTITY}
SERVER_MAC_ADDRESS = "AA:BB:CC:DD:EE:FF"


@pytest.fixture
def alloy():
    return MagicMock(spec=AlloyLogger)


@pytest.fixture
def wol_sender():
    return MagicMock(spec=WakeOnLanSender)


@pytest.fixture
def app(alloy, wol_sender):
    return create_app(
        server_mac_address=SERVER_MAC_ADDRESS,
        wol_sender=wol_sender,
        alloy_logger=alloy,
        bind_host="127.0.0.1",
    )


@pytest.fixture
def client(app):
    return app.test_client()


def test_wake_requires_identity_header(client):
    # Flask's test client defaults REMOTE_ADDR to 127.0.0.1, which would
    # now pass via the loopback exception -- pin a non-loopback address so
    # this test actually exercises the identity-header requirement.
    response = client.post("/wake", environ_base={"REMOTE_ADDR": "10.0.0.5"})

    assert response.status_code == 401


def test_wake_allows_a_loopback_caller_with_no_identity_header(client, wol_sender):
    response = client.post("/wake", environ_base={"REMOTE_ADDR": "127.0.0.1"})

    assert response.status_code == 200
    wol_sender.send.assert_called_once_with(SERVER_MAC_ADDRESS)


def test_wake_still_honors_identity_header_from_a_non_loopback_caller(client, wol_sender):
    response = client.post("/wake", headers=AUTH_HEADERS, environ_base={"REMOTE_ADDR": "10.0.0.5"})

    assert response.status_code == 200
    wol_sender.send.assert_called_once_with(SERVER_MAC_ADDRESS)


def test_wake_sends_magic_packet_to_configured_mac(client, wol_sender):
    client.post("/wake", headers=AUTH_HEADERS)

    wol_sender.send.assert_called_once_with(SERVER_MAC_ADDRESS)


def test_wake_returns_200_on_success(client):
    response = client.post("/wake", headers=AUTH_HEADERS)

    assert response.status_code == 200
    assert response.get_json() == {"wake": "succeeded"}


def test_wake_logs_requested_then_succeeded_with_caller_identity(client, alloy):
    client.post("/wake", headers=AUTH_HEADERS)

    assert alloy.send_event.call_args_list == [
        ((), {"event_type": "wake_requested", "outcome": "requested", "identity": CALLER_IDENTITY}),
        ((), {"event_type": "wake_succeeded", "outcome": "succeeded", "identity": CALLER_IDENTITY}),
    ]


def test_wake_returns_500_and_logs_failed_when_sender_raises(client, alloy, wol_sender):
    wol_sender.send.side_effect = OSError("network is unreachable")

    response = client.post("/wake", headers=AUTH_HEADERS)

    assert response.status_code == 500
    assert alloy.send_event.call_args_list[-1] == (
        (),
        {"event_type": "wake_failed", "outcome": "failed", "identity": CALLER_IDENTITY},
    )


def test_wake_does_not_guard_against_repeated_requests(client, wol_sender):
    # The API always attempts the action — the UI's disabled state is the only guard.
    client.post("/wake", headers=AUTH_HEADERS)
    client.post("/wake", headers=AUTH_HEADERS)

    assert wol_sender.send.call_count == 2


def test_create_app_refuses_off_tailnet_bind_host(alloy, wol_sender):
    with pytest.raises(BindOffTailnetError):
        create_app(
            server_mac_address=SERVER_MAC_ADDRESS,
            wol_sender=wol_sender,
            alloy_logger=alloy,
            bind_host="0.0.0.0",
        )


def test_create_app_rejects_a_malformed_mac_address_at_startup(alloy, wol_sender):
    # Fails fast at startup rather than mid-request, so every /wake call
    # can rely on the MAC already being valid (see app.py's build_magic_packet call).
    with pytest.raises(ValueError):
        create_app(
            server_mac_address="not-a-mac",
            wol_sender=wol_sender,
            alloy_logger=alloy,
            bind_host="127.0.0.1",
        )
