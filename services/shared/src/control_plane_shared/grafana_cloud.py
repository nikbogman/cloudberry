"""Structured event shipping straight to Grafana Cloud's Loki endpoint.

No local collector in front of this -- Grafana Cloud doesn't run one on
your behalf (Alloy is always self-hosted), and this repo has nothing else
worth standing up and maintaining a whole agent process for (ADR-0014).
Sends events as a Loki push-API payload (`{"streams": [...]}`) directly,
authenticated with the stack's own Loki basic-auth credentials.
`event_type`/`outcome`/`app` are stream labels; `identity`/`extra` ride in
the log line itself, since Loki labels are meant for low-cardinality
dimensions, not free-form identity strings.

A logging failure here must never break the caller's actual action, so
`send_event` swallows transport errors.
"""

import json
import logging
import time
from typing import Any

import requests

logger = logging.getLogger(__name__)


class GrafanaCloudLogger:
    def __init__(self, loki_url: str, loki_user: str, loki_api_key: str, app: str, timeout_seconds: float = 5.0):
        self._loki_url = loki_url
        self._auth = (loki_user, loki_api_key)
        self._app = app
        self._timeout_seconds = timeout_seconds

    def send_event(self, *, event_type: str, outcome: str, identity: str | None, extra: dict | None = None) -> None:
        line = json.dumps({"identity": identity, "extra": extra or {}})
        payload: dict[str, Any] = {
            "streams": [
                {
                    "stream": {"event_type": event_type, "outcome": outcome, "app": self._app},
                    "values": [[str(time.time_ns()), line]],
                }
            ]
        }
        try:
            requests.post(self._loki_url, json=payload, auth=self._auth, timeout=self._timeout_seconds)
        except requests.RequestException:
            logger.exception("failed to ship event %s to Grafana Cloud", event_type)
