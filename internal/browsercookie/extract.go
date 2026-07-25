// Package browsercookie extracts browser cookies for Canvas authentication.
package browsercookie

import (
	"context"
	"net/http"
	"runtime"
	"strings"

	"github.com/browserutils/kooky"
)

var sessionCookieNames = []string{
	"_instructure_session",
	"canvas_session",
}

const csrfCookieName = "_csrf_token"

// CookieReader abstracts cookie reading for testability.
type CookieReader interface {
	ReadCookies(ctx context.Context, filters ...kooky.Filter) (kooky.Cookies, error)
}

type DefaultReader struct{}

func (r *DefaultReader) ReadCookies(ctx context.Context, filters ...kooky.Filter) (kooky.Cookies, error) {
	return kooky.ReadCookies(ctx, filters...)
}

// Reader is the package-level cookie reader, overridable for dependency injection.
var Reader CookieReader = &DefaultReader{}

// scanCookies returns the Canvas session cookie ("name=value") and raw CSRF
func scanCookies(cookies kooky.Cookies) (sessionCookie, csrfToken string) {
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
	return sessionCookie, csrfToken
}

// ExtractCookies reads Canvas cookies for host. It filters by exact host match
func ExtractCookies(ctx context.Context, host string) (sessionCookie, csrfToken string, err error) {
	cookies, err := Reader.ReadCookies(ctx, kooky.Domain(host))
	if err != nil {
		return "", "", err
	}
	sessionCookie, csrfToken = scanCookies(cookies)
	if sessionCookie == "" {
		return "", csrfToken, ErrNoSessionCookie
	}
	return sessionCookie, csrfToken, nil
}

// IsSessionCookie reports whether cookie is a known Canvas session cookie.
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

// IsCSRFCookie reports whether cookie is the Canvas CSRF token cookie.
func IsCSRFCookie(cookie *http.Cookie) bool {
	if cookie == nil {
		return false
	}
	return cookie.Name == csrfCookieName
}

// ExtractCookiesForBrowser is like ExtractCookies but restricted to a single
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
	sessionCookie, csrfToken = scanCookies(cookies)
	if sessionCookie == "" {
		return "", csrfToken, ErrNoSessionCookie
	}
	return sessionCookie, csrfToken, nil
}

// AvailableBrowsers returns browser names supported on the current OS.
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
