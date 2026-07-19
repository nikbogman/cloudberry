"""Suspend-to-RAM: runs the configured system command as a subprocess. This
is the one external side effect in this app (per the spec's testing
conventions) — `SystemSuspender.suspend` is the seam mocked in app tests.

ADR-0002: suspend-to-RAM only. There is no code path here, or anywhere else
in this app, that could trigger a full shutdown (ACPI S5).
"""

import subprocess
from collections.abc import Sequence

DEFAULT_SUSPEND_COMMAND: Sequence[str] = ("systemctl", "suspend")


class SystemSuspender:
    """Suspends the host to RAM by running the configured command."""

    def __init__(self, command: Sequence[str] = DEFAULT_SUSPEND_COMMAND):
        if not command:
            raise ValueError("suspend command must not be empty")
        self._command = tuple(command)

    def suspend(self) -> None:
        subprocess.run(self._command, check=True)
