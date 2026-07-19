import flask
import pytest

from control_plane_shared.auth import get_caller_identity, require_tailnet_identity

IDENTITY_HEADER = "Tailscale-User-Login"


@pytest.fixture
def app():
    app = flask.Flask(__name__)

    @app.get("/protected")
    @require_tailnet_identity
    def protected():
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
