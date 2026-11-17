package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestContract exercises the full HTTP contract through the real router and a
// real HTTP server, so method handling, HEAD and 404/405 behave as in production.
func TestContract(t *testing.T) {
	orig := version
	version = "abc1234"
	defer func() { version = orig }()

	srv := httptest.NewServer(newRouter())
	defer srv.Close()

	tests := []struct {
		name   string
		method string
		path   string
		status int
		body   string // checked when not empty
		allow  string // checked when not empty
	}{
		{"root returns 42", http.MethodGet, "/", 200, "42", ""},
		{"query string is ignored", http.MethodGet, "/?x=1", 200, "42", ""},
		{"liveness", http.MethodGet, "/healthz", 200, "ok", ""},
		{"readiness", http.MethodGet, "/readyz", 200, "ready", ""},
		{"version", http.MethodGet, "/version", 200, "abc1234", ""},

		{"unknown path", http.MethodGet, "/missing", 404, "", ""},
		{"root does not match subpaths", http.MethodGet, "/42", 404, "", ""},
		{"trailing slash is a different path", http.MethodGet, "/healthz/", 404, "", ""},
		{"unknown path with other method", http.MethodPost, "/missing", 404, "", ""},

		{"POST root", http.MethodPost, "/", 405, "", "GET, HEAD"},
		{"PUT root", http.MethodPut, "/", 405, "", "GET, HEAD"},
		{"DELETE version", http.MethodDelete, "/version", 405, "", "GET, HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.status)
			}
			if tt.body != "" {
				if string(body) != tt.body {
					t.Errorf("body = %q, want %q", body, tt.body)
				}
				if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
					t.Errorf("Content-Type = %q", ct)
				}
			}
			if tt.allow != "" {
				if got := resp.Header.Get("Allow"); got != tt.allow {
					t.Errorf("Allow = %q, want %q", got, tt.allow)
				}
			}
		})
	}
}

// HEAD must succeed wherever GET does, with the same headers and no body.
func TestHeadHasNoBody(t *testing.T) {
	srv := httptest.NewServer(newRouter())
	defer srv.Close()

	for _, path := range []string{"/", "/healthz", "/readyz", "/version"} {
		resp, err := http.Head(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || len(body) != 0 {
			t.Errorf("HEAD %s: status = %d, body = %q; want 200 and no body", path, resp.StatusCode, body)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Errorf("HEAD %s: Content-Type = %q", path, ct)
		}
	}
}
