// Package browsercookie extracts browser cookies for Canvas authentication.
package browsercookie

import (
	"context"
	"net/http"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/chrome" // register Chrome cookie store finder
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

// StoreFinder abstracts cookie store discovery for testability.
type StoreFinder func(ctx context.Context) []kooky.CookieStore

// Finder is the package-level store finder; override in tests.
var Finder StoreFinder = kooky.FindAllCookieStores

// CookieReader abstracts cookie reading for testability (high-level mock).
type CookieReader interface {
	ReadCookies(ctx context.Context, filters ...kooky.Filter) (kooky.Cookies, error)
}

// Reader is the package-level reader; when non-nil it overrides Finder.
// Used only for testing.
var Reader CookieReader

// ExtractCookies reads browser cookies for the given host.
// It tries the default browser first, then falls back to Chrome.
// Stops at the first store that yields a valid session cookie.
// Returns sessionCookie in "name=value" format suitable for the HTTP Cookie header,
// and csrfToken as the raw value.
func ExtractCookies(ctx context.Context, host string) (sessionCookie, csrfToken string, err error) {
	// High-level mock path for testing.
	if Reader != nil {
		return extractWithReader(ctx, host)
	}

	domainFilter := kooky.Domain(host)

	// Try default browser first, then fall back to others.
	for _, browserName := range tryOrder() {
		for _, store := range Finder(ctx) {
			if store == nil {
				continue
			}
			if !strings.EqualFold(store.Browser(), browserName) {
				store.Close()
				continue
			}

			session, csrf := readStoreCookies(ctx, store, domainFilter)
			store.Close()

			if session != "" {
				return session, csrf, nil
			}
			if csrf != "" && csrfToken == "" {
				csrfToken = csrf
			}
		}
	}

	return "", csrfToken, ErrNoSessionCookie
}

// extractWithReader uses the mock Reader for testing.
func extractWithReader(ctx context.Context, host string) (sessionCookie, csrfToken string, err error) {
	cookies, err := Reader.ReadCookies(ctx, kooky.Domain(host))
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

// readStoreCookies reads cookies from a single store, returning the first
// session cookie and CSRF token found.
func readStoreCookies(ctx context.Context, store kooky.CookieStore, filters ...kooky.Filter) (sessionCookie, csrfToken string) {
	for cookie, err := range store.TraverseCookies(filters...) {
		if err != nil || cookie == nil {
			continue
		}
		if cookie.Name == csrfCookieName && csrfToken == "" {
			csrfToken = cookie.Value
		}
		if sessionCookie == "" {
			for _, sessionName := range sessionCookieNames {
				if cookie.Name == sessionName {
					sessionCookie = cookie.Name + "=" + cookie.Value
					break
				}
			}
		}
		if sessionCookie != "" {
			return sessionCookie, csrfToken
		}
	}
	return sessionCookie, csrfToken
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
// filtering to a specific browser by name (e.g. "chrome").
func ExtractCookiesForBrowser(ctx context.Context, host, browserName string) (sessionCookie, csrfToken string, err error) {
	domainFilter := kooky.Domain(host)

	for _, store := range Finder(ctx) {
		if store == nil {
			continue
		}
		if !strings.EqualFold(store.Browser(), browserName) {
			store.Close()
			continue
		}

		session, csrf := readStoreCookies(ctx, store, domainFilter)
		store.Close()

		if session != "" {
			return session, csrf, nil
		}
		if csrf != "" && csrfToken == "" {
			csrfToken = csrf
		}
	}

	return "", csrfToken, ErrNoSessionCookie
}

// AvailableBrowsers returns browser names available on the current OS for cookie extraction.
// Only returns browsers whose finders are compiled in (chrome on all platforms).
func AvailableBrowsers() []string {
	return []string{"chrome"}
}

// tryOrder returns the browser detection order: default browser first, then chrome as fallback.
func tryOrder() []string {
	if def := defaultBrowser(); def != "" && def != "chrome" {
		return []string{def, "chrome"}
	}
	return []string{"chrome"}
}
