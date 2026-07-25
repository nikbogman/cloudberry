// Wake-on-LAN: builds and broadcasts the magic packet that wakes the
// compute host. BuildMagicPacket is pure and tested directly;
// WakeOnLanSender.Send is the seam mocked in handler tests.
package gateway

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"syscall"
)

const (
	macRepetitions       = 16
	defaultBroadcastAddr = "255.255.255.255"
	defaultWakeOnLANPort = 9
)

// BuildMagicPacket builds a Wake-on-LAN magic packet for macAddress, which
// may be colon- or hyphen-separated and of either case.
func BuildMagicPacket(macAddress string) ([]byte, error) {
	cleaned := strings.NewReplacer(":", "", "-", "").Replace(macAddress)
	macBytes, err := hex.DecodeString(cleaned)
	if err != nil || len(macBytes) != 6 {
		return nil, fmt.Errorf("invalid MAC address: %q", macAddress)
	}

	packet := make([]byte, 0, 6+6*macRepetitions)
	for i := 0; i < 6; i++ {
		packet = append(packet, 0xff)
	}
	for i := 0; i < macRepetitions; i++ {
		packet = append(packet, macBytes...)
	}
	return packet, nil
}

// WakeOnLanSender broadcasts a WoL magic packet over UDP on the local L2 segment.
type WakeOnLanSender struct {
	broadcastAddress string
	port             int
}

// NewWakeOnLanSender returns a sender broadcasting to the default
// 255.255.255.255:9.
func NewWakeOnLanSender() *WakeOnLanSender {
	return &WakeOnLanSender{broadcastAddress: defaultBroadcastAddr, port: defaultWakeOnLANPort}
}

// Send builds and broadcasts a WoL magic packet for macAddress.
func (s *WakeOnLanSender) Send(macAddress string) error {
	packet, err := BuildMagicPacket(macAddress)
	if err != nil {
		return err
	}

	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return err
	}
	defer conn.Close()

	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		return fmt.Errorf("expected *net.UDPConn, got %T", conn)
	}
	rawConn, err := udpConn.SyscallConn()
	if err != nil {
		return err
	}
	var setsockoptErr error
	if err := rawConn.Control(func(fd uintptr) {
		setsockoptErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	}); err != nil {
		return err
	}
	if setsockoptErr != nil {
		return setsockoptErr
	}

	dest, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", s.broadcastAddress, s.port))
	if err != nil {
		return err
	}
	_, err = conn.WriteTo(packet, dest)
	return err
}
