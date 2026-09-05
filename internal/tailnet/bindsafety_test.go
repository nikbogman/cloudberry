package tailnet

import (
	"errors"
	"testing"
)

func TestAllowsLoopbackAndTailnetAddresses(t *testing.T) {
	hosts := []string{
		"127.0.0.1",
		"::1",
		"localhost",
		"100.64.0.1", // Tailscale CGNAT range (100.64.0.0/10)
		"100.100.100.100",
		"fd7a:115c:a1e0::1", // Tailscale IPv6 ULA range
	}
	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			if err := AssertTailnetOnlyBind(host); err != nil {
				t.Fatalf("AssertTailnetOnlyBind(%q) = %v, want nil", host, err)
			}
		})
	}
}

func TestRejectsOffTailnetAddresses(t *testing.T) {
	hosts := []string{
		"0.0.0.0",
		"192.168.1.50",
		"10.0.0.5",
		"8.8.8.8",
		"::",
	}
	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			err := AssertTailnetOnlyBind(host)
			var bindErr *BindOffTailnetError
			if !errors.As(err, &bindErr) {
				t.Fatalf("AssertTailnetOnlyBind(%q) = %v, want *BindOffTailnetError", host, err)
			}
		})
	}
}

func appFactory(host string) (string, error) {
	if err := AssertTailnetOnlyBind(host); err != nil {
		return "", err
	}
	return "app configured to bind " + host, nil
}

func TestAppFactoryRefusesToStartOnOffTailnetHost(t *testing.T) {
	_, err := appFactory("0.0.0.0")
	var bindErr *BindOffTailnetError
	if !errors.As(err, &bindErr) {
		t.Fatalf("got err %v, want *BindOffTailnetError", err)
	}
}

func TestAppFactoryStartsOnLoopbackHost(t *testing.T) {
	got, err := appFactory("127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "app configured to bind 127.0.0.1"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
