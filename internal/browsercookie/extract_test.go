package browsercookie

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/browserutils/kooky"
)

// MockCookieReader implements CookieReader for testing.
type MockCookieReader struct {
	cookies []*kooky.Cookie
	err     error
}

func (m *MockCookieReader) ReadCookies(ctx context.Context, filters ...kooky.Filter) (kooky.Cookies, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Apply filters to mock cookies.
	var result []*kooky.Cookie
	for _, cookie := range m.cookies {
		if cookie == nil {
			continue
		}
		pass := true
		for _, filter := range filters {
			if !filter.Filter(cookie) {
				pass = false
				break
			}
		}
		if pass {
			result = append(result, cookie)
		}
	}
	return result, nil
}

// Helper to create a test cookie.
func makeCookie(name, value, domain string) *kooky.Cookie {
	return &kooky.Cookie{
		Cookie: http.Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
		},
		Creation: time.Now(),
	}
}

func TestExtractCookies_ExactHost(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("_instructure_session", "session123", "canvas.school.edu"),
			makeCookie("_csrf_token", "csrf456", "canvas.school.edu"),
		},
	}
	Reader = mock

	session, csrf, err := ExtractCookies(context.Background(), "canvas.school.edu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "_instructure_session=session123" {
		t.Errorf("expected session '_instructure_session=session123', got '%s'", session)
	}
	if csrf != "csrf456" {
		t.Errorf("expected csrf 'csrf456', got '%s'", csrf)
	}
}

func TestExtractCookies_ParentDomain(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("_instructure_session", "session789", "school.edu"),
			makeCookie("_csrf_token", "csrf012", "school.edu"),
		},
	}
	Reader = mock

	session, csrf, err := ExtractCookies(context.Background(), "canvas.school.edu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "_instructure_session=session789" {
		t.Errorf("expected session '_instructure_session=session789', got '%s'", session)
	}
	if csrf != "csrf012" {
		t.Errorf("expected csrf 'csrf012', got '%s'", csrf)
	}
}

func TestExtractCookies_CanvasSession(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("canvas_session", "canvas123", "canvas.school.edu"),
			makeCookie("_csrf_token", "csrf456", "canvas.school.edu"),
		},
	}
	Reader = mock

	session, csrf, err := ExtractCookies(context.Background(), "canvas.school.edu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "canvas_session=canvas123" {
		t.Errorf("expected session 'canvas_session=canvas123', got '%s'", session)
	}
	if csrf != "csrf456" {
		t.Errorf("expected csrf 'csrf456', got '%s'", csrf)
	}
}

func TestExtractCookies_NoSessionCookie(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("_csrf_token", "csrf456", "canvas.school.edu"),
		},
	}
	Reader = mock

	_, _, err := ExtractCookies(context.Background(), "canvas.school.edu")
	if err != ErrNoSessionCookie {
		t.Errorf("expected ErrNoSessionCookie, got %v", err)
	}
}

func TestExtractCookies_NoCSRFToken(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("_instructure_session", "session123", "canvas.school.edu"),
		},
	}
	Reader = mock

	session, csrf, err := ExtractCookies(context.Background(), "canvas.school.edu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "_instructure_session=session123" {
		t.Errorf("expected session '_instructure_session=session123', got '%s'", session)
	}
	if csrf != "" {
		t.Errorf("expected empty csrf, got '%s'", csrf)
	}
}

func TestExtractCookies_MultipleDomains(t *testing.T) {
	// Test that we collect cookies from both exact host and parent domain.
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("_instructure_session", "session123", "canvas.school.edu"),
			makeCookie("_csrf_token", "csrf456", "school.edu"),
		},
	}
	Reader = mock

	session, csrf, err := ExtractCookies(context.Background(), "canvas.school.edu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "_instructure_session=session123" {
		t.Errorf("expected session '_instructure_session=session123', got '%s'", session)
	}
	if csrf != "csrf456" {
		t.Errorf("expected csrf 'csrf456', got '%s'", csrf)
	}
}

func TestExtractCookies_UnknownSessionCookieIgnored(t *testing.T) {
	// Unknown cookie names should be ignored.
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("some_other_cookie", "value123", "canvas.school.edu"),
			makeCookie("_csrf_token", "csrf456", "canvas.school.edu"),
		},
	}
	Reader = mock

	_, _, err := ExtractCookies(context.Background(), "canvas.school.edu")
	if err != ErrNoSessionCookie {
		t.Errorf("expected ErrNoSessionCookie, got %v", err)
	}
}

func TestExtractCookies_ParentDomainExtraction(t *testing.T) {
	tests := []struct {
		host     string
		expected string
	}{
		{"canvas.school.edu", "school.edu"},
		{"school.edu", "school.edu"},
		{"my.canvas.school.edu", "school.edu"},
		{"localhost", "localhost"},
		{"canvas.school.edu:8080", "school.edu"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := extractParentDomain(tt.host)
			if result != tt.expected {
				t.Errorf("extractParentDomain(%q) = %q, want %q", tt.host, result, tt.expected)
			}
		})
	}
}

func TestIsSessionCookie(t *testing.T) {
	tests := []struct {
		name     string
		cookie   *http.Cookie
		expected bool
	}{
		{"instructure_session", &http.Cookie{Name: "_instructure_session"}, true},
		{"canvas_session", &http.Cookie{Name: "canvas_session"}, true},
		{"other_cookie", &http.Cookie{Name: "other_cookie"}, false},
		{"nil_cookie", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSessionCookie(tt.cookie)
			if result != tt.expected {
				t.Errorf("IsSessionCookie() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCSRFCookie(t *testing.T) {
	tests := []struct {
		name     string
		cookie   *http.Cookie
		expected bool
	}{
		{"csrf_token", &http.Cookie{Name: "_csrf_token"}, true},
		{"other_cookie", &http.Cookie{Name: "other_cookie"}, false},
		{"nil_cookie", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCSRFCookie(tt.cookie)
			if result != tt.expected {
				t.Errorf("IsCSRFCookie() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractCookies_EmptyHost(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{},
	}
	Reader = mock

	_, _, err := ExtractCookies(context.Background(), "")
	if err != ErrNoSessionCookie {
		t.Errorf("expected ErrNoSessionCookie, got %v", err)
	}
}
