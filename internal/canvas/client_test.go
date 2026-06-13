package canvas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClientSetsFields(t *testing.T) {
	c := NewClient("https://canvas.example.com", "tok123", "0.1.0", 10*time.Second, 0)

	if c.baseURL != "https://canvas.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://canvas.example.com")
	}
	if c.token != "tok123" {
		t.Errorf("token = %q, want %q", c.token, "tok123")
	}
	if c.version != "0.1.0" {
		t.Errorf("version = %q, want %q", c.version, "0.1.0")
	}
	if c.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if c.httpClient.Timeout != 10*time.Second {
		t.Errorf("timeout = %v, want %v", c.httpClient.Timeout, 10*time.Second)
	}
	expectedUA := "canvas-cli/0.1.0 (+https://github.com/thedavidweng/canvas-cli)"
	if c.userAgent != expectedUA {
		t.Errorf("userAgent = %q, want %q", c.userAgent, expectedUA)
	}
}

func TestNewClientStripsTrailingSlash(t *testing.T) {
	c := NewClient("https://canvas.example.com/", "tok", "0.1.0", 5*time.Second, 0)
	if c.baseURL != "https://canvas.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://canvas.example.com")
	}
}

func TestNewClientStripsMultipleTrailingSlashes(t *testing.T) {
	c := NewClient("https://canvas.example.com///", "tok", "0.1.0", 5*time.Second, 0)
	if c.baseURL != "https://canvas.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://canvas.example.com")
	}
}

func TestDoSetsDefaultHeaders(t *testing.T) {
	var gotAuth, gotUA, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "mytoken", "1.0.0", 5*time.Second, 0)
	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer mytoken" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer mytoken")
	}
	expectedUA := "canvas-cli/1.0.0 (+https://github.com/thedavidweng/canvas-cli)"
	if gotUA != expectedUA {
		t.Errorf("User-Agent = %q, want %q", gotUA, expectedUA)
	}
	wantAccept := "application/json+canvas-string-ids"
	if gotAccept != wantAccept {
		t.Errorf("Accept = %q, want %q", gotAccept, wantAccept)
	}
}

func TestDoJoinsPathWithoutDoubleSlash(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	tests := []struct {
		name string
		path string
		want string
	}{
		{"leading slash", "/api/v1/courses", "/api/v1/courses"},
		{"no leading slash", "api/v1/courses", "/api/v1/courses"},
		{"base with path", "/api/v1/courses/1/assignments", "/api/v1/courses/1/assignments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.Do(context.Background(), http.MethodGet, tt.path, nil, nil)
			if err != nil {
				t.Fatalf("Do() error: %v", err)
			}
			resp.Body.Close()
			if gotPath != tt.want {
				t.Errorf("path = %q, want %q", gotPath, tt.want)
			}
		})
	}
}

func TestDoPassesQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	q := url.Values{"per_page": {"50"}, "page": {"2"}}
	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", q, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "50" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "50")
	}
	if parsed.Get("page") != "2" {
		t.Errorf("page = %q, want %q", parsed.Get("page"), "2")
	}
}

func TestDoContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 10*time.Second, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Do(ctx, http.MethodGet, "/api/v1/courses", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestDoRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 50*time.Millisecond, 0)

	_, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestDoSendsBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		r.Body.Read(b)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	body := strings.NewReader(`{"name":"test"}`)
	resp, err := c.Do(context.Background(), http.MethodPost, "/api/v1/courses", nil, body)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()

	if gotBody != `{"name":"test"}` {
		t.Errorf("body = %q, want %q", gotBody, `{"name":"test"}`)
	}
}

func TestDoMethodIsPassedThrough(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, m := range methods {
		resp, err := c.Do(context.Background(), m, "/api/v1/test", nil, nil)
		if err != nil {
			t.Fatalf("Do(%s) error: %v", m, err)
		}
		resp.Body.Close()
		if gotMethod != m {
			t.Errorf("method = %q, want %q", gotMethod, m)
		}
	}
}
