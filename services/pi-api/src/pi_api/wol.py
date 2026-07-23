"""Wake-on-LAN: builds and broadcasts the magic packet that wakes the main
server. `build_magic_packet` is pure and tested directly;
`WakeOnLanSender.send` is the seam mocked in app tests.
"""

import socket

MAGIC_PACKET_HEADER = b"\xff" * 6
MAC_REPETITIONS = 16
WOL_PORT = 9


def build_magic_packet(mac_address: str) -> bytes:
    mac_bytes = bytes.fromhex(mac_address.replace(":", "").replace("-", ""))
    if len(mac_bytes) != 6:
        raise ValueError(f"invalid MAC address: {mac_address!r}")
    return MAGIC_PACKET_HEADER + mac_bytes * MAC_REPETITIONS


class WakeOnLanSender:
    """Broadcasts a WoL magic packet over UDP on the local L2 segment."""

    def __init__(self, broadcast_address: str = "255.255.255.255", port: int = WOL_PORT):
        self._broadcast_address = broadcast_address
        self._port = port

    def send(self, mac_address: str) -> None:
        packet = build_magic_packet(mac_address)
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
            sock.sendto(packet, (self._broadcast_address, self._port))
