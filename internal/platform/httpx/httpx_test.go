package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureTraceID_passThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Trace-Id", "trace-from-client")
	got := EnsureTraceID(req)
	if got != "trace-from-client" {
		t.Fatalf("expected trace passthrough, got %q", got)
	}
}

func TestEnsureTraceID_generate(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	got := EnsureTraceID(req)
	if len(got) != 16 {
		t.Fatalf("expected 16 hex chars, got %q len=%d", got, len(got))
	}
}

func TestJSON_envelopeAndHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusOK, "abc123", 0, "ok", map[string]string{"k": "v"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get("X-Trace-Id"); got != "abc123" {
		t.Fatalf("header trace=%q", got)
	}

	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || body.Message != "ok" || body.TraceID != "abc123" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
