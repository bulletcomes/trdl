package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootReturns42(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	rootHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "42" {
		t.Fatalf("body = %q, want %q", got, "42")
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
}

func TestUnknownPathReturns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	rootHandler(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUnsupportedMethodReturns405(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	rootHandler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", got)
	}
}

func TestHealthAndReadiness(t *testing.T) {
	tests := []struct {
		name string
		path string
		fn   http.HandlerFunc
		body string
	}{
		{"health", "/healthz", healthHandler, "ok"},
		{"readiness", "/readyz", readyHandler, "ready"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			tt.fn(rec, req)
			if rec.Code != http.StatusOK || rec.Body.String() != tt.body {
				t.Fatalf("got status=%d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestVersion(t *testing.T) {
	orig := version
	version = "1.2.3"
	defer func() { version = orig }()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	versionHandler(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "1.2.3" {
		t.Fatalf("got status=%d body=%q", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/version", nil)
	rec = httptest.NewRecorder()
	versionHandler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
