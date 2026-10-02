package waker

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestBuildMagicPacketHasSixLeadingFFBytes(t *testing.T) {
	packet, err := BuildMagicPacket("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := bytes.Repeat([]byte{0xff}, 6)
	if !bytes.Equal(packet[:6], want) {
		t.Fatalf("got %x, want %x", packet[:6], want)
	}
}

func TestBuildMagicPacketRepeatsMacSixteenTimes(t *testing.T) {
	packet, err := BuildMagicPacket("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	macBytes, _ := hex.DecodeString("AABBCCDDEEFF")
	want := bytes.Repeat(macBytes, 16)
	if !bytes.Equal(packet[6:], want) {
		t.Fatalf("got %x, want %x", packet[6:], want)
	}
	if len(packet) != 102 {
		t.Fatalf("got packet length %d, want 102", len(packet))
	}
}

func TestBuildMagicPacketAcceptsHyphenSeparatedMac(t *testing.T) {
	hyphen, err := BuildMagicPacket("AA-BB-CC-DD-EE-FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	colon, err := BuildMagicPacket("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(hyphen, colon) {
		t.Fatalf("got %x, want %x", hyphen, colon)
	}
}

func TestBuildMagicPacketAcceptsLowercaseMac(t *testing.T) {
	lower, err := BuildMagicPacket("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	upper, err := BuildMagicPacket("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(lower, upper) {
		t.Fatalf("got %x, want %x", lower, upper)
	}
}

func TestBuildMagicPacketRejectsInvalidMac(t *testing.T) {
	invalid := []string{
		"not-a-mac",
		"AA:BB:CC:DD:EE",       // too short
		"AA:BB:CC:DD:EE:FF:00", // too long
		"",
	}
	for _, mac := range invalid {
		t.Run(mac, func(t *testing.T) {
			if _, err := BuildMagicPacket(mac); err == nil {
				t.Fatalf("BuildMagicPacket(%q) succeeded, want error", mac)
			}
		})
	}
}
