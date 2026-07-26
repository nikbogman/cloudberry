package tailnet

import (
	"fmt"
	"net"
	"net/netip"
)

var (
	tailscaleIPv4Range = netip.MustParsePrefix("100.64.0.0/10")
	tailscaleIPv6Range = netip.MustParsePrefix("fd7a:115c:a1e0::/48")
)

type BindOffTailnetError struct {
	msg string
}

func (e *BindOffTailnetError) Error() string { return e.msg }

// AssertTailnetOnlyBind allows loopback and Tailscale's own address ranges
// (the CGNAT IPv4 range it assigns nodes, and its IPv6 ULA range) --
// nothing else.
func AssertTailnetOnlyBind(host string) error {
	if host == "localhost" {
		return nil
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		ips, lookupErr := net.LookupHost(host)
		if lookupErr != nil || len(ips) == 0 {
			return &BindOffTailnetError{msg: fmt.Sprintf("could not resolve host %q to check its bind safety", host)}
		}
		addr, err = netip.ParseAddr(ips[0])
		if err != nil {
			return &BindOffTailnetError{msg: fmt.Sprintf("could not resolve host %q to check its bind safety", host)}
		}
	}

	if addr.IsLoopback() {
		return nil
	}
	if tailscaleIPv4Range.Contains(addr) || tailscaleIPv6Range.Contains(addr) {
		return nil
	}

	return &BindOffTailnetError{
		msg: fmt.Sprintf(
			"refusing to bind to %q: not loopback or a tailnet address, "+
				"which would expose this control-plane app off-tailnet", host,
		),
	}
}
