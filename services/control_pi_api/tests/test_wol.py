import pytest

from control_pi_api.wol import build_magic_packet


def test_build_magic_packet_has_six_leading_ff_bytes():
    packet = build_magic_packet("AA:BB:CC:DD:EE:FF")

    assert packet[:6] == b"\xff" * 6


def test_build_magic_packet_repeats_mac_sixteen_times():
    packet = build_magic_packet("AA:BB:CC:DD:EE:FF")
    mac_bytes = bytes.fromhex("AABBCCDDEEFF")

    assert packet[6:] == mac_bytes * 16
    assert len(packet) == 102


def test_build_magic_packet_accepts_hyphen_separated_mac():
    packet = build_magic_packet("AA-BB-CC-DD-EE-FF")

    assert packet == build_magic_packet("AA:BB:CC:DD:EE:FF")


def test_build_magic_packet_accepts_lowercase_mac():
    packet = build_magic_packet("aa:bb:cc:dd:ee:ff")

    assert packet == build_magic_packet("AA:BB:CC:DD:EE:FF")


@pytest.mark.parametrize(
    "mac_address",
    [
        "not-a-mac",
        "AA:BB:CC:DD:EE",  # too short
        "AA:BB:CC:DD:EE:FF:00",  # too long
        "",
    ],
)
def test_build_magic_packet_rejects_invalid_mac(mac_address):
    with pytest.raises(ValueError):
        build_magic_packet(mac_address)
