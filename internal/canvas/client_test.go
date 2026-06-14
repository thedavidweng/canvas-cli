package canvas

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

// --- Session cookie auth tests (Steps 1.3-1.7) ---

func TestDo_CookieAuth_SendsCookieHeader(t *testing.T) {
	var gotCookie, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("my-session-cookie", "")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()

	if gotCookie != "my-session-cookie" {
		t.Errorf("Cookie = %q, want %q", gotCookie, "my-session-cookie")
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty (cookie auth, no token)", gotAuth)
	}
}

func TestDo_TokenTakesPrecedenceOverCookie(t *testing.T) {
	var gotCookie, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-token", "0.1.0", 5*time.Second, 0).WithCookie("my-cookie", "my-csrf")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer my-token")
	}
	if gotCookie != "" {
		t.Errorf("Cookie = %q, want empty (token takes precedence)", gotCookie)
	}
}

func TestDo_CookieAuth_CSRFHeaderForUnsafeMethods(t *testing.T) {
	var gotCSRF string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCSRF = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "my-csrf-token")

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, m := range methods {
		gotCSRF = ""
		resp, err := c.Do(context.Background(), m, "/api/v1/test", nil, nil)
		if err != nil {
			t.Fatalf("Do(%s) error: %v", m, err)
		}
		resp.Body.Close()
		if gotCSRF != "my-csrf-token" {
			t.Errorf("Do(%s): X-CSRF-Token = %q, want %q", m, gotCSRF, "my-csrf-token")
		}
	}
}

func TestDo_CookieAuth_NoCSRFHeaderForSafeMethods(t *testing.T) {
	var gotCSRF string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCSRF = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "my-csrf-token")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()
	if gotCSRF != "" {
		t.Errorf("X-CSRF-Token = %q, want empty for GET", gotCSRF)
	}
}

func TestDo_CookieAuth_MissingCSRF_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "")

	_, err := c.Do(context.Background(), http.MethodPost, "/api/v1/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing CSRF token with cookie auth on POST")
	}
}

func TestDo_CookieAuth_CachesCSRFTokenFromResponse(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("X-CSRF-Token", "server-csrf-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "")

	// First GET (safe method) - no CSRF needed, but response should cache it
	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("first Do() error: %v", err)
	}
	resp.Body.Close()

	// Second POST (unsafe method) - should use cached CSRF
	var gotCSRF string
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCSRF = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusOK)
	})

	resp, err = c.Do(context.Background(), http.MethodPost, "/api/v1/test", nil, nil)
	if err != nil {
		t.Fatalf("second Do() error: %v", err)
	}
	resp.Body.Close()

	if gotCSRF != "server-csrf-token" {
		t.Errorf("X-CSRF-Token = %q, want cached 'server-csrf-token'", gotCSRF)
	}
}

func TestDoURL_CookieAuth_SendsCookieHeader(t *testing.T) {
	var gotCookie, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("my-session-cookie", "")

	resp, err := c.DoURL(context.Background(), http.MethodGet, srv.URL+"/api/v1/courses", nil)
	if err != nil {
		t.Fatalf("DoURL() error: %v", err)
	}
	resp.Body.Close()

	if gotCookie != "my-session-cookie" {
		t.Errorf("Cookie = %q, want %q", gotCookie, "my-session-cookie")
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty", gotAuth)
	}
}

func TestDoURL_TokenTakesPrecedence(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-token", "0.1.0", 5*time.Second, 0).WithCookie("my-cookie", "")

	resp, err := c.DoURL(context.Background(), http.MethodGet, srv.URL+"/api/v1/courses", nil)
	if err != nil {
		t.Fatalf("DoURL() error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer my-token")
	}
}

func TestDo_TokenAuth_NoCookieHeader(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-token", "0.1.0", 5*time.Second, 0)

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()

	if gotCookie != "" {
		t.Errorf("Cookie = %q, want empty (token auth)", gotCookie)
	}
}

func TestDo_CookieAuth_CSRFWithCachedFromPreviousResponse(t *testing.T) {
	// First server returns CSRF in response header
	firstSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", "cached-csrf")
		w.WriteHeader(http.StatusOK)
	}))
	defer firstSrv.Close()

	// Second server checks the CSRF is sent
	var gotCSRF string
	secondSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCSRF = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer secondSrv.Close()

	c := NewClient(firstSrv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "")

	// GET to first server to cache CSRF
	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("first Do() error: %v", err)
	}
	resp.Body.Close()

	// Override baseURL to second server, POST should use cached CSRF
	c.baseURL = secondSrv.URL
	resp, err = c.Do(context.Background(), http.MethodPost, "/api/v1/test", nil, nil)
	if err != nil {
		t.Fatalf("second Do() error: %v", err)
	}
	resp.Body.Close()

	if gotCSRF != "cached-csrf" {
		t.Errorf("X-CSRF-Token = %q, want cached 'cached-csrf'", gotCSRF)
	}
}

// --- Redirect classification tests (Step 1.7) ---

func TestDo_CookieAuth_AuthRedirect_ReturnsSessionExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://school.instructure.com/login")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "")

	_, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err == nil {
		t.Fatal("expected error for auth redirect")
	}
	if !func() bool { var e *CookieSessionExpiredError; return errors.As(err, &e) }() {
		t.Errorf("expected CookieSessionExpiredError, got %T: %v", err, err)
	}
}

func TestDo_CookieAuth_NonAuthRedirect_FollowsForGet(t *testing.T) {
	finalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer finalSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", finalSrv.URL+"/api/v1/courses")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (should follow non-auth redirect)", resp.StatusCode, http.StatusOK)
	}
}

func TestDo_CookieAuth_NonAuthRedirect_StripHeadersOnFollow(t *testing.T) {
	var gotCookieOnFollow, gotAuthOnFollow, gotCSRFOnFollow string
	finalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookieOnFollow = r.Header.Get("Cookie")
		gotAuthOnFollow = r.Header.Get("Authorization")
		gotCSRFOnFollow = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer finalSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", finalSrv.URL+"/api/v1/courses")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "csrf")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	if gotCookieOnFollow != "" {
		t.Errorf("Cookie on follow = %q, want empty (should be stripped)", gotCookieOnFollow)
	}
	if gotAuthOnFollow != "" {
		t.Errorf("Authorization on follow = %q, want empty (should be stripped)", gotAuthOnFollow)
	}
	if gotCSRFOnFollow != "" {
		t.Errorf("X-CSRF-Token on follow = %q, want empty (should be stripped)", gotCSRFOnFollow)
	}
}

func TestDo_CookieAuth_UnsafeMethodRedirect_DoesNotReplay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a redirect to a different origin
		w.Header().Set("Location", "https://other-school.instructure.com/api/v1/test")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "csrf")

	resp, err := c.Do(context.Background(), http.MethodPost, "/api/v1/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	// For unsafe methods, the client should NOT follow cross-origin redirects.
	// The response should be the 302 itself.
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status = %d, want %d (should not follow cross-origin redirect for POST)", resp.StatusCode, http.StatusFound)
	}
}

func TestDo_TokenAuth_RedirectHandledNormally(t *testing.T) {
	finalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer finalSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", finalSrv.URL+"/api/v1/courses")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	// Token auth should follow redirects normally (Go default behavior)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (token auth should follow redirect)", resp.StatusCode, http.StatusOK)
	}
}

func TestDo_CookieAuth_AuthRedirect_SamlPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://school.instructure.com/saml/sso")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "")

	_, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err == nil {
		t.Fatal("expected error for SAML redirect")
	}
	if !func() bool { var e *CookieSessionExpiredError; return errors.As(err, &e) }() {
		t.Errorf("expected CookieSessionExpiredError, got %T: %v", err, err)
	}
}

// --- Phase 6: Integration tests ---

// Step 6.1: Full cookie auth flow.
// Mock Canvas server at /api/v1/users/self returns 200 with user JSON when
// Cookie header is present (and no Authorization).
func TestIntegration_CookieAuth_FullFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/self" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Cookie") == "" {
			t.Error("expected Cookie header, got none")
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header with cookie auth, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"42","name":"Test User","login_id":"test@example.com"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc123", "")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/users/self", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var user struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		LoginID string `json:"login_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if user.ID != "42" {
		t.Errorf("user ID = %q, want %q", user.ID, "42")
	}
	if user.Name != "Test User" {
		t.Errorf("user Name = %q, want %q", user.Name, "Test User")
	}
	if user.LoginID != "test@example.com" {
		t.Errorf("user LoginID = %q, want %q", user.LoginID, "test@example.com")
	}
}

// Step 6.2: Full cookie auth expiry flow.
// Mock server returns 401. Verify error path returns the 401 response.
func TestIntegration_CookieAuth_ExpiryFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"Invalid access token"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=expired", "")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/users/self", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Verify the caller can detect session expiry via IsCookieSessionExpired.
	bodyBytes, _ := io.ReadAll(resp.Body)
	if !IsCookieSessionExpired(resp, bodyBytes, srv.URL) {
		t.Error("expected IsCookieSessionExpired to return true for 401 with cookie auth")
	}
}

// Step 6.3: CSRF from response header.
// GET /api/v1/courses returns 200 with X-CSRF-Token response header.
// POST /api/v1/courses/1/assignments expects X-CSRF-Token header.
// Verify: GET caches the CSRF token, POST uses it.
func TestIntegration_CSRF_FromResponseHeader(t *testing.T) {
	var postCSRF string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/courses":
			w.Header().Set("X-CSRF-Token", "server-issued-csrf")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id":"1","name":"CS 101"}]`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/courses/1/assignments":
			postCSRF = r.Header.Get("X-CSRF-Token")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"99","name":"Homework 1"}`))

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc", "")

	// GET caches the CSRF token from the response header.
	getResp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	getResp.Body.Close()

	// POST should use the cached CSRF token.
	postBody := strings.NewReader(`{"name":"Homework 1"}`)
	postResp, err := c.Do(context.Background(), http.MethodPost, "/api/v1/courses/1/assignments", nil, postBody)
	if err != nil {
		t.Fatalf("POST error: %v", err)
	}
	postResp.Body.Close()

	if postCSRF != "server-issued-csrf" {
		t.Errorf("POST X-CSRF-Token = %q, want %q (should be cached from GET response)", postCSRF, "server-issued-csrf")
	}
}

// Step 6.3: If POST is attempted without any CSRF source, returns error
// before hitting the server.
func TestIntegration_CSRF_Missing_ReturnsErrorBeforeServer(t *testing.T) {
	var serverHit bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc", "")

	_, err := c.Do(context.Background(), http.MethodPost, "/api/v1/courses", nil, strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected error for missing CSRF token on POST")
	}
	if serverHit {
		t.Error("server should not be hit when CSRF token is missing")
	}
	if !strings.Contains(err.Error(), "CSRF") {
		t.Errorf("error = %q, want it to contain 'CSRF'", err.Error())
	}
}

// Step 6.4: Redirect to /login → session expired error.
func TestIntegration_Redirect_LoginPath_SessionExpired(t *testing.T) {
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", srvURL+"/login")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	srvURL = srv.URL

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc", "")

	_, err := c.Do(context.Background(), http.MethodGet, "/api/v1/courses", nil, nil)
	if err == nil {
		t.Fatal("expected session expired error for redirect to /login")
	}
	if !func() bool { var e *CookieSessionExpiredError; return errors.As(err, &e) }() {
		t.Errorf("expected CookieSessionExpiredError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "session expired") {
		t.Errorf("error = %q, want it to contain 'session expired'", err.Error())
	}
}

// Step 6.4: Redirect to S3 → followed with credentials stripped.
func TestIntegration_Redirect_S3_FollowsWithCredentialsStripped(t *testing.T) {
	var followCookie, followAuth, followCSRF string

	finalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		followCookie = r.Header.Get("Cookie")
		followAuth = r.Header.Get("Authorization")
		followCSRF = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("file content"))
	}))
	defer finalSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", finalSrv.URL+"/bucket/file.pdf")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc", "my-csrf")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/files/1", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d (should follow S3 redirect)", resp.StatusCode, http.StatusOK)
	}
	if followCookie != "" {
		t.Errorf("Cookie on follow = %q, want empty (should be stripped)", followCookie)
	}
	if followAuth != "" {
		t.Errorf("Authorization on follow = %q, want empty (should be stripped)", followAuth)
	}
	if followCSRF != "" {
		t.Errorf("X-CSRF-Token on follow = %q, want empty (should be stripped)", followCSRF)
	}
}

// Step 6.4: POST redirect to external URL → NOT followed (unsafe method).
func TestIntegration_Redirect_UnsafeMethod_NotFollowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://external.example.com/api/v1/submit")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc", "csrf-token")

	resp, err := c.Do(context.Background(), http.MethodPost, "/api/v1/submit", nil, strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	// For unsafe methods, the client should NOT follow cross-origin redirects.
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status = %d, want %d (should not follow redirect for POST)", resp.StatusCode, http.StatusFound)
	}
}

// Step 6.4: Redirect to Shibboleth IdP host → session expired.
func TestIntegration_Redirect_ShibbolethIdP_SessionExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://shibboleth.school.edu/sso/SAML2/POST")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc", "")

	_, err := c.Do(context.Background(), http.MethodGet, "/api/v1/data", nil, nil)
	if err == nil {
		t.Fatal("expected session expired error for redirect to Shibboleth IdP")
	}
	if !func() bool { var e *CookieSessionExpiredError; return errors.As(err, &e) }() {
		t.Errorf("expected CookieSessionExpiredError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "session expired") {
		t.Errorf("error = %q, want it to contain 'session expired'", err.Error())
	}
}

// Step 6.5: Token takes precedence over cookie.
// Create client with both token AND cookie set.
// Verify request sends Authorization header (not Cookie).
func TestIntegration_TokenPrecedence_OverCookie(t *testing.T) {
	var gotAuth, gotCookie string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"42","name":"Test User"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-api-token", "0.1.0", 5*time.Second, 0).WithCookie("canvas_session=abc", "csrf")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/users/self", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer my-api-token" {
		t.Errorf("Authorization = %q, want %q (token should take precedence)", gotAuth, "Bearer my-api-token")
	}
	if gotCookie != "" {
		t.Errorf("Cookie = %q, want empty (token takes precedence, cookie should not be sent)", gotCookie)
	}
}

// Integration: cookie auth through doWithRetry path (retries on 5xx).
func TestIntegration_CookieAuth_RetryOn5xx(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"42","name":"Recovered User"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 1).WithCookie("canvas_session=abc", "")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/users/self", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (1 retry after 500)", callCount)
	}
}

// Integration: cookie auth does NOT retry on 401.
func TestIntegration_CookieAuth_NoRetryOn401(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 2).WithCookie("canvas_session=abc", "")

	resp, err := c.Do(context.Background(), http.MethodGet, "/api/v1/users/self", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (401 should not trigger retry)", callCount)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
