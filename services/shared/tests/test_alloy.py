import json

import pytest
import responses
from requests.exceptions import ConnectionError as RequestsConnectionError

from control_plane_shared.alloy import AlloyLogger

PUSH_URL = "https://alloy.example.internal/loki/api/v1/push"


@pytest.fixture
def logger():
    return AlloyLogger(push_url=PUSH_URL)


@responses.activate
def test_send_event_posts_structured_payload_to_alloy(logger):
    responses.post(PUSH_URL, status=204)

    logger.send_event(event_type="wake_requested", outcome="success", identity="nicola@example.com")

    assert len(responses.calls) == 1
    request = responses.calls[0].request
    assert request.url == PUSH_URL
    body = json.loads(request.body)
    assert body["event_type"] == "wake_requested"
    assert body["outcome"] == "success"
    assert body["identity"] == "nicola@example.com"


@responses.activate
def test_send_event_includes_extra_fields(logger):
    responses.post(PUSH_URL, status=204)

    logger.send_event(
        event_type="reachability_changed",
        outcome="reachable",
        identity=None,
        extra={"previous_state": "unreachable"},
    )

    body = json.loads(responses.calls[0].request.body)
    assert body["extra"] == {"previous_state": "unreachable"}


@responses.activate
def test_send_event_does_not_raise_when_alloy_is_unreachable(logger):
    responses.post(PUSH_URL, body=RequestsConnectionError("no route to host"))

    # A logging failure must never take down the caller's request path.
    logger.send_event(event_type="wake_requested", outcome="success", identity="nicola@example.com")

    assert len(responses.calls) == 1
