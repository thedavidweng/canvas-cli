package canvas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// NormalizeError converts an HTTP error response into a structured Envelope.
// If baseURL is provided (variadic), cookie session expiry is checked and
// overrides the error when detected.
func NormalizeError(resp *http.Response, command string, baseURL ...string) Envelope {
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	errInfo := normalizeErrorInfo(resp, bodyBytes, baseURL...)

	return Envelope{
		OK:    false,
		Error: errInfo,
		Meta: Meta{
			SchemaVersion: SchemaVersion,
			Command:       command,
		},
	}
}

// NormalizeErrorFromBody creates an ErrorInfo from an HTTP response whose body
// has already been read. Use this when the body bytes are needed for other
// purposes (e.g. JSON envelope construction) before error processing.
// If baseURL is provided (variadic), cookie session expiry is checked.
func NormalizeErrorFromBody(resp *http.Response, bodyBytes []byte, baseURL ...string) ErrorInfo {
	return *normalizeErrorInfo(resp, bodyBytes, baseURL...)
}

// normalizeErrorInfo is the shared implementation for NormalizeError and
// NormalizeErrorFromBody. If baseURL is provided, cookie session expiry is
// checked and overrides the error when detected.
func normalizeErrorInfo(resp *http.Response, bodyBytes []byte, baseURL ...string) *ErrorInfo {
	var bodyMap map[string]any
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &bodyMap)
	}

	errInfo := &ErrorInfo{
		Status:       resp.StatusCode,
		ResponseBody: bodyMap,
	}

	// Extract message from body if available, with fallback.
	if bodyMap != nil {
		if msg, ok := bodyMap["message"].(string); ok {
			errInfo.Message = msg
		}
	}
	if errInfo.Message == "" {
		errInfo.Message = http.StatusText(resp.StatusCode)
	}

	// Map status codes to error codes, categories, and retryable flag.
	switch resp.StatusCode {
	case http.StatusUnauthorized: // 401
		errInfo.Code = "CANVAS_AUTH_ERROR"
		errInfo.Category = "auth"
	case http.StatusForbidden: // 403
		errInfo.Code = "CANVAS_PERMISSION_DENIED"
		errInfo.Category = "permission"
		// 403 with rate limit exhausted is retryable.
		if resp.Header.Get("X-Rate-Limit-Remaining") == "0" {
			errInfo.Code = "CANVAS_RATE_LIMIT"
			errInfo.Category = "rate_limit"
			errInfo.Retryable = true
		}
	case http.StatusNotFound: // 404
		errInfo.Code = "CANVAS_NOT_FOUND"
		errInfo.Category = "not_found"
	case http.StatusUnprocessableEntity: // 422
		errInfo.Code = "CANVAS_VALIDATION_ERROR"
		errInfo.Category = "validation"
	case http.StatusTooManyRequests: // 429
		errInfo.Code = "CANVAS_RATE_LIMIT"
		errInfo.Category = "rate_limit"
		errInfo.Retryable = true
	default:
		if resp.StatusCode >= 500 {
			errInfo.Code = "CANVAS_SERVER_ERROR"
			errInfo.Category = "server"
			errInfo.Retryable = true
		} else {
			errInfo.Code = "CANVAS_API_ERROR"
			errInfo.Category = "api"
		}
	}

	if reqID := resp.Header.Get("X-Request-Id"); reqID != "" {
		errInfo.CanvasRequestID = reqID
	}

	// Check for cookie session expiry if baseURL is provided.
	if len(baseURL) > 0 && baseURL[0] != "" && IsCookieSessionExpired(resp, bodyBytes, baseURL[0]) {
		errInfo.Code = "CANVAS_SESSION_EXPIRED"
		errInfo.Message = "session expired. Re-authenticate: canvas auth login"
		errInfo.Category = "auth"
	}

	return errInfo
}

// CookieSessionExpiredError indicates that the session cookie is no longer
// valid and the user needs to re-authenticate.
type CookieSessionExpiredError struct {
	Location string // The redirect URL that triggered the detection.
}

func (e *CookieSessionExpiredError) Error() string {
	return fmt.Sprintf("session expired (redirected to %s); please re-authenticate", e.Location)
}

// IsCookieSessionExpired checks whether an HTTP response indicates that the
// cookie session has expired and the user needs to re-authenticate.
func IsCookieSessionExpired(resp *http.Response, bodyBytes []byte, baseURL string) bool {
	// Normal JSON success responses are not auth failures.
	if resp.StatusCode == 200 {
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(ct, "application/json") {
			return false
		}
	}

	// 404 is not an auth failure.
	if resp.StatusCode == 404 {
		return false
	}

	bodyLower := strings.ToLower(string(bodyBytes))

	// 401 is always an auth failure.
	if resp.StatusCode == 401 {
		return true
	}

	// 403 with auth/session/CSRF signal in body.
	if resp.StatusCode == 403 {
		if strings.Contains(bodyLower, "csrf") ||
			strings.Contains(bodyLower, "session") ||
			strings.Contains(bodyLower, "authenticity") {
			return true
		}
	}

	// 302/303 redirect to auth pages.
	if resp.StatusCode == 302 || resp.StatusCode == 303 {
		location := resp.Header.Get("Location")
		if isAuthRedirect(location) {
			return true
		}
		// External host with auth path prefix.
		if !hostMatches(baseURL, location) && hasAuthPathPrefix(location) {
			return true
		}
	}

	// 200 with HTML when JSON was expected (login page served instead of API).
	if resp.StatusCode == 200 {
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(ct, "text/html") {
			if resp.Request != nil {
				accept := resp.Request.Header.Get("Accept")
				if strings.Contains(strings.ToLower(accept), "application/json") {
					return true
				}
			}
		}
	}

	// 422 with CSRF authenticity error string (Canvas returns this for
	// invalid authenticity token on form submissions).
	if resp.StatusCode == 422 {
		if strings.Contains(bodyLower, "authenticity token") || strings.Contains(bodyLower, "csrf") {
			return true
		}
	}

	return false
}

// hostMatches returns true if the host portions of two URLs match.
func hostMatches(baseURL, locationURL string) bool {
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return false
	}
	loc, err := url.Parse(locationURL)
	if err != nil || loc.Host == "" {
		return false
	}
	return strings.EqualFold(base.Host, loc.Host)
}

// hasAuthPathPrefix returns true if the URL path starts with a common
// authentication path prefix.
func hasAuthPathPrefix(location string) bool {
	u, err := url.Parse(location)
	if err != nil {
		return false
	}
	path := strings.ToLower(u.Path)
	authPrefixes := []string{
		"/login", "/auth", "/sso", "/cas", "/saml", "/idp", "/shibboleth",
		"/signin", "/sign-in",
	}
	for _, prefix := range authPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// isAuthRedirect returns true if the redirect location indicates an
// authentication page (login, SSO, CAS, Shibboleth, etc.).
func isAuthRedirect(location string) bool {
	u, err := url.Parse(location)
	if err != nil {
		return false
	}

	path := strings.ToLower(u.Path)
	host := strings.ToLower(u.Host)

	// Check path prefixes that indicate auth redirects.
	authPaths := []string{
		"/login", "/logout", "/saml", "/cas", "/shibboleth.sso/", "/idp/",
	}
	for _, prefix := range authPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	// Check host patterns for SSO infrastructure.
	if strings.Contains(host, ".shibboleth.") || strings.HasPrefix(host, "shibboleth.") ||
		strings.Contains(host, ".cas.") || strings.HasPrefix(host, "cas.") {
		return true
	}

	return false
}
