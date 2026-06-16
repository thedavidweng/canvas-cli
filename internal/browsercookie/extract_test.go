package browsercookie

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
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

// mockBrowserInfo implements kooky.BrowserInfo for testing.
type mockBrowserInfo struct {
	name string
}

func (b *mockBrowserInfo) Browser() string        { return b.name }
func (b *mockBrowserInfo) Profile() string        { return "" }
func (b *mockBrowserInfo) IsDefaultProfile() bool { return true }
func (b *mockBrowserInfo) FilePath() string       { return "" }

// makeCookieWithBrowser creates a test cookie with BrowserInfo set.
func makeCookieWithBrowser(name, value, domain, browserName string) *kooky.Cookie {
	c := makeCookie(name, value, domain)
	if browserName != "" {
		c.Browser = &mockBrowserInfo{name: browserName}
	}
	return c
}

// --- ExtractCookiesForBrowser tests ---

func TestExtractCookiesForBrowser_MatchingBrowser(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookieWithBrowser("_instructure_session", "chrome-sess", "canvas.school.edu", "chrome"),
			makeCookieWithBrowser("_csrf_token", "chrome-csrf", "canvas.school.edu", "chrome"),
		},
	}
	Reader = mock

	session, csrf, err := ExtractCookiesForBrowser(context.Background(), "canvas.school.edu", "chrome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "_instructure_session=chrome-sess" {
		t.Errorf("expected session '_instructure_session=chrome-sess', got %q", session)
	}
	if csrf != "chrome-csrf" {
		t.Errorf("expected csrf 'chrome-csrf', got %q", csrf)
	}
}

func TestExtractCookiesForBrowser_NonMatchingBrowser(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookieWithBrowser("_instructure_session", "firefox-sess", "canvas.school.edu", "firefox"),
			makeCookieWithBrowser("_csrf_token", "firefox-csrf", "canvas.school.edu", "firefox"),
		},
	}
	Reader = mock

	_, _, err := ExtractCookiesForBrowser(context.Background(), "canvas.school.edu", "chrome")
	if err != ErrNoSessionCookie {
		t.Errorf("expected ErrNoSessionCookie, got %v", err)
	}
}

func TestExtractCookiesForBrowser_CaseInsensitiveBrowserMatch(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookieWithBrowser("_instructure_session", "sess", "canvas.school.edu", "Chrome"),
			makeCookieWithBrowser("_csrf_token", "csrf", "canvas.school.edu", "Chrome"),
		},
	}
	Reader = mock

	session, _, err := ExtractCookiesForBrowser(context.Background(), "canvas.school.edu", "chrome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "_instructure_session=sess" {
		t.Errorf("expected session '_instructure_session=sess', got %q", session)
	}
}

func TestExtractCookiesForBrowser_NoSessionCookie(t *testing.T) {
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookieWithBrowser("_csrf_token", "csrf456", "canvas.school.edu", "chrome"),
		},
	}
	Reader = mock

	_, csrf, err := ExtractCookiesForBrowser(context.Background(), "canvas.school.edu", "chrome")
	if err != ErrNoSessionCookie {
		t.Errorf("expected ErrNoSessionCookie, got %v", err)
	}
	if csrf != "csrf456" {
		t.Errorf("expected csrf 'csrf456', got %q", csrf)
	}
}

func TestExtractCookiesForBrowser_ErrorFromReader(t *testing.T) {
	mock := &MockCookieReader{
		err: fmt.Errorf("browser locked"),
	}
	Reader = mock

	_, _, err := ExtractCookiesForBrowser(context.Background(), "canvas.school.edu", "chrome")
	if err == nil {
		t.Fatal("expected error from reader")
	}
	if err.Error() != "browser locked" {
		t.Errorf("expected 'browser locked', got %v", err)
	}
}

func TestExtractCookiesForBrowser_NilBrowserInfo(t *testing.T) {
	// Cookies without BrowserInfo should be rejected by the browser filter.
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookie("_instructure_session", "sess", "canvas.school.edu"),
		},
	}
	Reader = mock

	_, _, err := ExtractCookiesForBrowser(context.Background(), "canvas.school.edu", "chrome")
	if err != ErrNoSessionCookie {
		t.Errorf("expected ErrNoSessionCookie for cookie without BrowserInfo, got %v", err)
	}
}

func TestExtractCookiesForBrowser_MultipleBrowsers(t *testing.T) {
	// Cookies from different browsers; only chrome should match.
	mock := &MockCookieReader{
		cookies: []*kooky.Cookie{
			makeCookieWithBrowser("_instructure_session", "firefox-sess", "canvas.school.edu", "firefox"),
			makeCookieWithBrowser("_instructure_session", "chrome-sess", "canvas.school.edu", "chrome"),
			makeCookieWithBrowser("_csrf_token", "chrome-csrf", "canvas.school.edu", "chrome"),
		},
	}
	Reader = mock

	session, csrf, err := ExtractCookiesForBrowser(context.Background(), "canvas.school.edu", "chrome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "_instructure_session=chrome-sess" {
		t.Errorf("expected chrome session, got %q", session)
	}
	if csrf != "chrome-csrf" {
		t.Errorf("expected chrome csrf, got %q", csrf)
	}
}

// --- AvailableBrowsers tests ---

func TestAvailableBrowsers_ReturnsNonEmpty(t *testing.T) {
	browsers := AvailableBrowsers()
	if len(browsers) == 0 {
		t.Error("expected non-empty browser list")
	}
}

func TestAvailableBrowsers_ContainsExpectedBrowsers(t *testing.T) {
	browsers := AvailableBrowsers()
	browserSet := make(map[string]bool)
	for _, b := range browsers {
		browserSet[b] = true
	}

	// All platforms should have at least chrome and firefox.
	for _, expected := range []string{"chrome", "firefox"} {
		if !browserSet[expected] {
			t.Errorf("expected %q in available browsers for %s, got %v", expected, runtime.GOOS, browsers)
		}
	}
}

func TestAvailableBrowsers_PlatformSpecific(t *testing.T) {
	browsers := AvailableBrowsers()
	browserSet := make(map[string]bool)
	for _, b := range browsers {
		browserSet[b] = true
	}

	switch runtime.GOOS {
	case "darwin":
		if !browserSet["safari"] {
			t.Error("expected 'safari' on darwin")
		}
		if !browserSet["edge"] {
			t.Error("expected 'edge' on darwin")
		}
	case "linux":
		if !browserSet["chromium"] {
			t.Error("expected 'chromium' on linux")
		}
	case "windows":
		if !browserSet["edge"] {
			t.Error("expected 'edge' on windows")
		}
	}
}

// --- ReadCookies (DefaultReader) tests ---

func TestReadCookies_DefaultReader_DoesNotPanic(t *testing.T) {
	// Save and restore the original Reader.
	origReader := Reader
	defer func() { Reader = origReader }()

	// DefaultReader.ReadCookies calls kooky.ReadCookies which tries to find
	// real browser cookie stores. It should return an error (no stores registered
	// in test) but must not panic.
	Reader = &DefaultReader{}
	_, _, err := ExtractCookies(context.Background(), "example.com")
	// We don't care about the specific error; just that it didn't panic.
	_ = err
}
