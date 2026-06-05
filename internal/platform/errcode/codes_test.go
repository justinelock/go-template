package errcode

import "testing"

func TestCodes_matchDocumentation(t *testing.T) {
	cases := map[string]int{
		"OK":                      OK,
		"TokenRequired":           TokenRequired,
		"TokenInvalid":            TokenInvalid,
		"DownstreamUnavailable":   DownstreamUnavailable,
		"WebSocketNotImplemented": WebSocketNotImplemented,
	}
	want := map[string]int{
		"OK":                      0,
		"TokenRequired":           40101,
		"TokenInvalid":            40102,
		"DownstreamUnavailable":   50002,
		"WebSocketNotImplemented": 50101,
	}
	for name, code := range cases {
		if code != want[name] {
			t.Fatalf("%s: got %d want %d", name, code, want[name])
		}
	}
}
