"""Structured event shipping to Grafana Alloy.

Both Control APIs are fully stateless — Alloy is the only place wake,
suspend, and reachability history lives. A logging failure here must never
break the caller's actual action, so `send_event` swallows transport errors.
"""

import logging

import requests

logger = logging.getLogger(__name__)


class AlloyLogger:
    def __init__(self, push_url: str, timeout_seconds: float = 5.0):
        self._push_url = push_url
        self._timeout_seconds = timeout_seconds

    def send_event(self, *, event_type: str, outcome: str, identity: str | None, extra: dict | None = None) -> None:
        payload = {
            "event_type": event_type,
            "outcome": outcome,
            "identity": identity,
            "extra": extra or {},
        }
        try:
            requests.post(self._push_url, json=payload, timeout=self._timeout_seconds)
        except requests.RequestException:
            logger.exception("failed to ship event %s to Grafana Alloy", event_type)
