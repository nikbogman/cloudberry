import subprocess
from unittest.mock import MagicMock

import pytest

from control_server_api.suspend import DEFAULT_SUSPEND_COMMAND, SystemSuspender


def test_suspend_runs_the_default_command(monkeypatch):
    run_mock = MagicMock()
    monkeypatch.setattr(subprocess, "run", run_mock)

    SystemSuspender().suspend()

    run_mock.assert_called_once_with(tuple(DEFAULT_SUSPEND_COMMAND), check=True)


def test_default_command_is_systemctl_suspend_only():
    # Suspend-to-RAM only — never a full shutdown (ACPI S5).
    assert tuple(DEFAULT_SUSPEND_COMMAND) == ("systemctl", "suspend")


def test_suspend_runs_a_configured_command(monkeypatch):
    run_mock = MagicMock()
    monkeypatch.setattr(subprocess, "run", run_mock)

    SystemSuspender(command=["some-other-suspend-tool", "--now"]).suspend()

    run_mock.assert_called_once_with(("some-other-suspend-tool", "--now"), check=True)


def test_suspend_propagates_called_process_error(monkeypatch):
    monkeypatch.setattr(
        subprocess, "run", MagicMock(side_effect=subprocess.CalledProcessError(1, DEFAULT_SUSPEND_COMMAND))
    )

    with pytest.raises(subprocess.CalledProcessError):
        SystemSuspender().suspend()


def test_suspend_propagates_oserror_when_command_is_not_found(monkeypatch):
    monkeypatch.setattr(subprocess, "run", MagicMock(side_effect=FileNotFoundError("no such file")))

    with pytest.raises(OSError):
        SystemSuspender().suspend()


def test_constructor_rejects_an_empty_command():
    # Fails fast at startup rather than mid-request.
    with pytest.raises(ValueError):
        SystemSuspender(command=[])
