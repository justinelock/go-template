package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequiresAuth_routes(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/v1/auth/login", false},
		{"/v1/auth/register", false},
		{"/v1/auth/logout", true},
		{"/v1/member/users/profile", true},
		{"/v1/order/orders", true},
		{"/v1/order/orders/123", true},
		{"/healthz", false},
	}
	for _, tc := range cases {
		if got := RequiresAuth(tc.path); got != tc.want {
			t.Fatalf("path %s: want %v got %v", tc.path, tc.want, got)
		}
	}
}

func TestExtractToken_priority(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/logout?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	req.Header.Set("token", "header-token")
	if got := ExtractToken(req); got != "bearer-token" {
		t.Fatalf("bearer first: got %q", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/auth/logout?token=query-token", nil)
	req2.Header.Set("token", "header-token")
	if got := ExtractToken(req2); got != "header-token" {
		t.Fatalf("header second: got %q", got)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/v1/auth/logout?token=query-token", nil)
	if got := ExtractToken(req3); got != "query-token" {
		t.Fatalf("query third: got %q", got)
	}
}
