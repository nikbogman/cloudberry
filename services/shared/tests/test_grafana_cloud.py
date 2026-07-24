import json

import pytest
import responses
from requests.exceptions import ConnectionError as RequestsConnectionError

from control_plane_shared.grafana_cloud import GrafanaCloudLogger

LOKI_URL = "https://logs-prod-000.grafana.net/loki/api/v1/push"
LOKI_USER = "123456"
LOKI_API_KEY = "glc_faketoken"


@pytest.fixture
def logger():
    return GrafanaCloudLogger(loki_url=LOKI_URL, loki_user=LOKI_USER, loki_api_key=LOKI_API_KEY, app="pi-api")


@responses.activate
def test_send_event_posts_a_loki_push_payload_with_basic_auth(logger):
    responses.post(LOKI_URL, status=204)

    logger.send_event(event_type="wake_requested", outcome="success", identity="nicola@example.com")

    assert len(responses.calls) == 1
    request = responses.calls[0].request
    assert request.url == LOKI_URL
    assert request.headers["Authorization"].startswith("Basic ")

    body = json.loads(request.body)
    stream = body["streams"][0]
    assert stream["stream"] == {"event_type": "wake_requested", "outcome": "success", "app": "pi-api"}

    [[timestamp, line]] = stream["values"]
    assert timestamp.isdigit()
    assert json.loads(line) == {"identity": "nicola@example.com", "extra": {}}


@responses.activate
def test_send_event_includes_extra_fields_in_the_line(logger):
    responses.post(LOKI_URL, status=204)

    logger.send_event(
        event_type="reachability_changed",
        outcome="reachable",
        identity=None,
        extra={"previous_state": "unreachable"},
    )

    body = json.loads(responses.calls[0].request.body)
    [[_, line]] = body["streams"][0]["values"]
    assert json.loads(line) == {"identity": None, "extra": {"previous_state": "unreachable"}}


@responses.activate
def test_send_event_does_not_raise_when_grafana_cloud_is_unreachable(logger):
    responses.post(LOKI_URL, body=RequestsConnectionError("no route to host"))

    # A logging failure must never take down the caller's request path.
    logger.send_event(event_type="wake_requested", outcome="success", identity="nicola@example.com")

    assert len(responses.calls) == 1
