import flask
import pytest

from control_plane_shared.auth import (
    get_caller_identity,
    require_tailnet_identity,
    require_tailnet_identity_or_loopback,
)

IDENTITY_HEADER = "Tailscale-User-Login"


@pytest.fixture
def app():
    app = flask.Flask(__name__)

    @app.get("/protected")
    @require_tailnet_identity
    def protected():
        return {"identity": get_caller_identity()}

    @app.get("/loopback-protected")
    @require_tailnet_identity_or_loopback
    def loopback_protected():
        return {"identity": get_caller_identity()}

    return app


@pytest.fixture
def client(app):
    return app.test_client()


def test_rejects_request_with_no_identity_header(client):
    response = client.get("/protected")

    assert response.status_code == 401


def test_rejects_request_with_blank_identity_header(client):
    response = client.get("/protected", headers={IDENTITY_HEADER: "   "})

    assert response.status_code == 401


def test_allows_request_with_valid_identity_header(client):
    response = client.get("/protected", headers={IDENTITY_HEADER: "nicola@example.com"})

    assert response.status_code == 200


def test_get_caller_identity_returns_header_value(client):
    response = client.get("/protected", headers={IDENTITY_HEADER: "nicola@example.com"})

    assert response.get_json() == {"identity": "nicola@example.com"}


def test_loopback_allows_a_request_from_127_0_0_1_with_no_identity_header(client):
    response = client.get("/loopback-protected", environ_base={"REMOTE_ADDR": "127.0.0.1"})

    assert response.status_code == 200


def test_loopback_rejects_a_request_from_elsewhere_with_no_identity_header(client):
    response = client.get("/loopback-protected", environ_base={"REMOTE_ADDR": "10.0.0.5"})

    assert response.status_code == 401


def test_loopback_still_honors_a_valid_identity_header_from_elsewhere(client):
    response = client.get(
        "/loopback-protected",
        headers={IDENTITY_HEADER: "nicola@example.com"},
        environ_base={"REMOTE_ADDR": "10.0.0.5"},
    )

    assert response.status_code == 200
    assert response.get_json() == {"identity": "nicola@example.com"}
