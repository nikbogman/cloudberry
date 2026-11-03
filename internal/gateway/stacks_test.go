package gateway

import (
	"strings"
	"testing"
)

func TestRouteFromCompose(t *testing.T) {
	const svc = `"services":{"server":{"ports":[{"host_ip":"10.0.0.2","published":"2283"}]},"db":{}}`
	for in, want := range map[string]string{
		`{"name":"immich",` + svc + `,"x-gateway":{"service":"server","wake":true}}`: "immich http://10.0.0.2:2283 wake",
		`{"name":"plain",` + svc + `}`:                                                                   "not routed",
		`{"name":"x",` + svc + `,"x-gateway":{}}`:                                                        "error: must name",
		`{"name":"x",` + svc + `,"x-gateway":{"service":"db"}}`:                                          "error: LAN IP",
		`{"name":"x","services":{"a":{"ports":[{"host_ip":"0.0.0.0","published":"1"}]}},"x-gateway":{}}`: "error: LAN IP",
		`{"name":"one","services":{"a":{"ports":[{"host_ip":"10.0.0.2","published":"3000"}]}},"x-gateway":{"path":"/one","port":8444}}`: "one http://10.0.0.2:3000",
	} {
		r, ok, err := RouteFromCompose([]byte(in))
		var got string
		switch {
		case err != nil:
			got = "error: " + err.Error()
		case !ok:
			got = "not routed"
		default:
			got = r.Name + " " + r.Upstream
			if r.Wake {
				got += " wake"
			}
		}
		if !strings.HasPrefix(want, "error: ") && got != want ||
			strings.HasPrefix(want, "error: ") && !strings.Contains(got, strings.TrimPrefix(want, "error: ")) {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
}
