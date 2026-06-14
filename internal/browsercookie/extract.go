// Package browsercookie extracts browser cookies for Canvas authentication.
package browsercookie

import (
	"context"
	"net/http"
	"runtime"
	"strings"

	"github.com/browserutils/kooky"
)

// KOOKY_AVAILABLE indicates whether browser cookie extraction is compiled in.
// Always true when the kooky dependency is present.
const KOOKY_AVAILABLE = true

// Known session cookie names for Canvas LMS.
var sessionCookieNames = []string{
	"_instructure_session",
	"canvas_session",
}

// CSRF cookie name.
const csrfCookieName = "_csrf_token"

// CookieReader abstracts cookie reading for testability.
type CookieReader interface {
	ReadCookies(ctx context.Context, filters ...kooky.Filter) (kooky.Cookies, error)
}

// DefaultReader uses kooky's built-in cookie reading.
type DefaultReader struct{}

func (r *DefaultReader) ReadCookies(ctx context.Context, filters ...kooky.Filter) (kooky.Cookies, error) {
	return kooky.ReadCookies(ctx, filters...)
}

// Package-level reader variable for dependency injection.
var Reader CookieReader = &DefaultReader{}

// ExtractCookies reads browser cookies for the given host.
// Returns sessionCookie in "name=value" format suitable for the HTTP Cookie header,
// and csrfToken as the raw value.
// Filters by exact host match only (no parent domain matching to avoid
// over-broad results with ccTLDs like *.ac.uk, *.edu.au, *.co.uk).
func ExtractCookies(ctx context.Context, host string) (sessionCookie, csrfToken string, err error) {
	cookies, err := Reader.ReadCookies(ctx, kooky.Domain(host))
	if err != nil {
		return "", "", err
	}

	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}

		name := cookie.Name
		value := cookie.Value

		// Check for CSRF token.
		if name == csrfCookieName && csrfToken == "" {
			csrfToken = value
			continue
		}

		// Check for session cookie by known names.
		if sessionCookie == "" {
			for _, sessionName := range sessionCookieNames {
				if name == sessionName {
					sessionCookie = name + "=" + value
					break
				}
			}
		}
	}

	// Return error if no session cookie found.
	if sessionCookie == "" {
		return "", csrfToken, ErrNoSessionCookie
	}

	return sessionCookie, csrfToken, nil
}

// IsSessionCookie checks if a cookie is a known session cookie.
func IsSessionCookie(cookie *http.Cookie) bool {
	if cookie == nil {
		return false
	}
	for _, name := range sessionCookieNames {
		if cookie.Name == name {
			return true
		}
	}
	return false
}

// IsCSRFCookie checks if a cookie is the CSRF token cookie.
func IsCSRFCookie(cookie *http.Cookie) bool {
	if cookie == nil {
		return false
	}
	return cookie.Name == csrfCookieName
}

// ExtractCookiesForBrowser reads browser cookies for the given host,
// filtering to a specific browser by name (e.g. "chrome", "firefox", "safari").
func ExtractCookiesForBrowser(ctx context.Context, host, browserName string) (sessionCookie, csrfToken string, err error) {
	browserFilter := kooky.FilterFunc(func(c *kooky.Cookie) bool {
		if c.Browser == nil {
			return false
		}
		return strings.EqualFold(c.Browser.Browser(), browserName)
	})

	cookies, err := Reader.ReadCookies(ctx, kooky.Domain(host), browserFilter)
	if err != nil {
		return "", "", err
	}

	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		if cookie.Name == csrfCookieName && csrfToken == "" {
			csrfToken = cookie.Value
			continue
		}
		if sessionCookie == "" {
			for _, sessionName := range sessionCookieNames {
				if cookie.Name == sessionName {
					sessionCookie = cookie.Name + "=" + cookie.Value
					break
				}
			}
		}
	}

	if sessionCookie == "" {
		return "", csrfToken, ErrNoSessionCookie
	}
	return sessionCookie, csrfToken, nil
}

// AvailableBrowsers returns browser names available on the current OS for cookie extraction.
func AvailableBrowsers() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"chrome", "firefox", "safari", "edge", "brave", "opera"}
	case "linux":
		return []string{"chrome", "firefox", "chromium", "opera", "brave"}
	case "windows":
		return []string{"chrome", "firefox", "edge", "brave", "opera"}
	default:
		return []string{"chrome", "firefox"}
	}
}
