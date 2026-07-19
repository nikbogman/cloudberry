import pytest

from control_plane_shared.bind_safety import BindOffTailnetError, assert_tailnet_only_bind


@pytest.mark.parametrize(
    "host",
    [
        "127.0.0.1",
        "::1",
        "localhost",
        "100.64.0.1",  # Tailscale CGNAT range (100.64.0.0/10)
        "100.100.100.100",
        "fd7a:115c:a1e0::1",  # Tailscale IPv6 ULA range
    ],
)
def test_allows_loopback_and_tailnet_addresses(host):
    assert_tailnet_only_bind(host)  # must not raise


@pytest.mark.parametrize(
    "host",
    [
        "0.0.0.0",
        "192.168.1.50",
        "10.0.0.5",
        "8.8.8.8",
        "::",
    ],
)
def test_rejects_off_tailnet_addresses(host):
    with pytest.raises(BindOffTailnetError):
        assert_tailnet_only_bind(host)


def create_app(host: str) -> str:
    """Stand-in for a Control API's app factory: refuses to start if `host`
    would expose it off-tailnet, matching how tickets 02-04 must call this
    at startup before binding Flask's dev/prod server to a host."""
    assert_tailnet_only_bind(host)
    return f"app configured to bind {host}"


def test_app_factory_refuses_to_start_on_off_tailnet_host():
    with pytest.raises(BindOffTailnetError):
        create_app("0.0.0.0")


def test_app_factory_starts_on_loopback_host():
    assert create_app("127.0.0.1") == "app configured to bind 127.0.0.1"
