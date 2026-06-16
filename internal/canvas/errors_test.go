package canvas

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func newResponse(status int, body string, headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNormalizeError401(t *testing.T) {
	resp := newResponse(401, `{"message":"Unauthorized"}`, nil)
	env := NormalizeError(resp, "courses.list")

	if env.OK {
		t.Error("OK should be false")
	}
	if env.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if env.Error.Code != "CANVAS_AUTH_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_AUTH_ERROR")
	}
	if env.Error.Category != "auth" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "auth")
	}
	if env.Error.Status != 401 {
		t.Errorf("Status = %d, want 401", env.Error.Status)
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for auth errors")
	}
	if env.Meta.Command != "courses.list" {
		t.Errorf("Command = %q, want %q", env.Meta.Command, "courses.list")
	}
}

func TestNormalizeError403(t *testing.T) {
	resp := newResponse(403, `{"message":"Forbidden"}`, nil)
	env := NormalizeError(resp, "courses.update")

	if env.Error.Code != "CANVAS_PERMISSION_DENIED" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_PERMISSION_DENIED")
	}
	if env.Error.Category != "permission" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "permission")
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for permission errors")
	}
}

func TestNormalizeError404(t *testing.T) {
	resp := newResponse(404, `{"message":"Not Found"}`, nil)
	env := NormalizeError(resp, "courses.get")

	if env.Error.Code != "CANVAS_NOT_FOUND" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_NOT_FOUND")
	}
	if env.Error.Category != "not_found" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "not_found")
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for not found errors")
	}
}

func TestNormalizeError422(t *testing.T) {
	resp := newResponse(422, `{"message":"Unprocessable Entity","errors":[{"message":"Name is required"}]}`, nil)
	env := NormalizeError(resp, "assignments.create")

	if env.Error.Code != "CANVAS_VALIDATION_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_VALIDATION_ERROR")
	}
	if env.Error.Category != "validation" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "validation")
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for validation errors")
	}
}

func TestNormalizeErrorCanvasRequestID(t *testing.T) {
	resp := newResponse(400, `{"message":"Bad Request"}`, map[string]string{
		"X-Request-Id": "req-abc-123",
	})
	env := NormalizeError(resp, "api.get")

	if env.Error.CanvasRequestID != "req-abc-123" {
		t.Errorf("CanvasRequestID = %q, want %q", env.Error.CanvasRequestID, "req-abc-123")
	}
}

func TestNormalizeErrorFromBody401(t *testing.T) {
	body := []byte(`{"message":"Unauthorized"}`)
	resp := newResponse(401, string(body), nil)

	errInfo := NormalizeErrorFromBody(resp, body)

	if errInfo.Status != 401 {
		t.Errorf("Status = %d, want 401", errInfo.Status)
	}
	if errInfo.Code != "CANVAS_AUTH_ERROR" {
		t.Errorf("Code = %q, want %q", errInfo.Code, "CANVAS_AUTH_ERROR")
	}
	if errInfo.Category != "auth" {
		t.Errorf("Category = %q, want %q", errInfo.Category, "auth")
	}
	if errInfo.Message != "Unauthorized" {
		t.Errorf("Message = %q, want %q", errInfo.Message, "Unauthorized")
	}
	if errInfo.Retryable {
		t.Error("Retryable should be false for auth errors")
	}
}

func TestNormalizeErrorFromBody404(t *testing.T) {
	body := []byte(`{"message":"Not Found"}`)
	resp := newResponse(404, string(body), nil)

	errInfo := NormalizeErrorFromBody(resp, body)

	if errInfo.Code != "CANVAS_NOT_FOUND" {
		t.Errorf("Code = %q, want %q", errInfo.Code, "CANVAS_NOT_FOUND")
	}
	if errInfo.Category != "not_found" {
		t.Errorf("Category = %q, want %q", errInfo.Category, "not_found")
	}
}

func TestNormalizeErrorFromBody429(t *testing.T) {
	body := []byte(`{"message":"Too Many Requests"}`)
	resp := newResponse(429, string(body), map[string]string{
		"X-Rate-Limit-Remaining": "0",
	})

	errInfo := NormalizeErrorFromBody(resp, body)

	if errInfo.Code != "CANVAS_RATE_LIMIT" {
		t.Errorf("Code = %q, want %q", errInfo.Code, "CANVAS_RATE_LIMIT")
	}
	if errInfo.Category != "rate_limit" {
		t.Errorf("Category = %q, want %q", errInfo.Category, "rate_limit")
	}
	if !errInfo.Retryable {
		t.Error("Retryable should be true for rate limit errors")
	}
}

func TestNormalizeErrorFromBody500(t *testing.T) {
	body := []byte(`{"message":"Internal Server Error"}`)
	resp := newResponse(500, string(body), nil)

	errInfo := NormalizeErrorFromBody(resp, body)

	if errInfo.Code != "CANVAS_SERVER_ERROR" {
		t.Errorf("Code = %q, want %q", errInfo.Code, "CANVAS_SERVER_ERROR")
	}
	if errInfo.Category != "server" {
		t.Errorf("Category = %q, want %q", errInfo.Category, "server")
	}
	if !errInfo.Retryable {
		t.Error("Retryable should be true for 5xx errors")
	}
}

func TestNormalizeErrorFromBody422(t *testing.T) {
	body := []byte(`{"message":"Unprocessable Entity"}`)
	resp := newResponse(422, string(body), nil)

	errInfo := NormalizeErrorFromBody(resp, body)

	if errInfo.Code != "CANVAS_VALIDATION_ERROR" {
		t.Errorf("Code = %q, want %q", errInfo.Code, "CANVAS_VALIDATION_ERROR")
	}
	if errInfo.Category != "validation" {
		t.Errorf("Category = %q, want %q", errInfo.Category, "validation")
	}
}

func TestNormalizeErrorFromBody403RateLimit(t *testing.T) {
	body := []byte(`{"message":"Rate Limit Exceeded"}`)
	resp := newResponse(403, string(body), map[string]string{
		"X-Rate-Limit-Remaining": "0",
	})

	errInfo := NormalizeErrorFromBody(resp, body)

	if errInfo.Code != "CANVAS_RATE_LIMIT" {
		t.Errorf("Code = %q, want %q", errInfo.Code, "CANVAS_RATE_LIMIT")
	}
	if errInfo.Category != "rate_limit" {
		t.Errorf("Category = %q, want %q", errInfo.Category, "rate_limit")
	}
	if !errInfo.Retryable {
		t.Error("Retryable should be true for rate limit errors")
	}
}

func TestNormalizeErrorFromBodyWithRequestID(t *testing.T) {
	body := []byte(`{"message":"Bad Request"}`)
	resp := newResponse(400, string(body), map[string]string{
		"X-Request-Id": "req-xyz-789",
	})

	errInfo := NormalizeErrorFromBody(resp, body)

	if errInfo.CanvasRequestID != "req-xyz-789" {
		t.Errorf("CanvasRequestID = %q, want %q", errInfo.CanvasRequestID, "req-xyz-789")
	}
}

func TestNormalizeErrorBodyPreserved(t *testing.T) {
	body := `{"message":"Bad Request","errors":[{"message":"invalid field"}]}`
	resp := newResponse(400, body, nil)
	env := NormalizeError(resp, "api.get")

	if env.Error.ResponseBody == nil {
		t.Fatal("ResponseBody should not be nil")
	}
	bodyMap, ok := env.Error.ResponseBody.(map[string]any)
	if !ok {
		t.Fatalf("ResponseBody should be a map, got %T", env.Error.ResponseBody)
	}
	if bodyMap["message"] != "Bad Request" {
		t.Errorf("ResponseBody[message] = %v, want %q", bodyMap["message"], "Bad Request")
	}
}

func TestNormalizeError500(t *testing.T) {
	resp := newResponse(500, `{"message":"Internal Server Error"}`, nil)
	env := NormalizeError(resp, "api.get")

	if env.Error.Code != "CANVAS_SERVER_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_SERVER_ERROR")
	}
	if env.Error.Category != "server" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "server")
	}
	if !env.Error.Retryable {
		t.Error("Retryable should be true for 5xx errors")
	}
}

func TestNormalizeErrorGenericCode(t *testing.T) {
	resp := newResponse(409, `{"message":"Conflict"}`, nil)
	env := NormalizeError(resp, "api.get")

	if env.Error.Code != "CANVAS_API_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_API_ERROR")
	}
	if env.Error.Category != "api" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "api")
	}
}

// --- Cookie session expiry tests ---

func TestIsCookieSessionExpiredErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "cookie session expired error",
			err:  &CookieSessionExpiredError{Location: "https://school.instructure.com/login"},
			want: true,
		},
		{
			name: "other error",
			err:  fmt.Errorf("some other error"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e *CookieSessionExpiredError
			got := errors.As(tt.err, &e)
			if got != tt.want {
				t.Errorf("errors.As(CookieSessionExpiredError) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCookieSessionExpiredError_Error(t *testing.T) {
	err := &CookieSessionExpiredError{Location: "https://school.instructure.com/login"}
	msg := err.Error()
	if !strings.Contains(msg, "session expired") {
		t.Errorf("error message should contain 'session expired', got: %s", msg)
	}
	if !strings.Contains(msg, "/login") {
		t.Errorf("error message should contain location, got: %s", msg)
	}
}

func TestClassifyRedirect_AuthRedirects(t *testing.T) {
	tests := []struct {
		name     string
		location string
		isAuth   bool
	}{
		{"login path", "https://school.instructure.com/login", true},
		{"logout path", "https://school.instructure.com/logout", true},
		{"saml path", "https://school.instructure.com/saml/sso", true},
		{"cas path", "https://school.instructure.com/cas/login", true},
		{"shibboleth path", "https://school.instructure.com/Shibboleth.sso/Login", true},
		{"idp path", "https://school.instructure.com/idp/SSO", true},
		{"shibboleth host", "https://shibboleth.school.edu/sso", true},
		{"cas host", "https://cas.school.edu/login", true},
		{"regular API path", "https://school.instructure.com/api/v1/courses", false},
		{"homepage", "https://school.instructure.com/", false},
		{"dashboard", "https://school.instructure.com/dashboard", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAuthRedirect(tt.location)
			if got != tt.isAuth {
				t.Errorf("isAuthRedirect(%q) = %v, want %v", tt.location, got, tt.isAuth)
			}
		})
	}
}

func TestIsCookieSessionExpired(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		headers    map[string]string
		baseURL    string
		reqHeaders map[string]string
		want       bool
	}{
		// 401 is always expired.
		{
			name:   "401 unauthorized",
			status: 401,
			body:   `{"message":"Unauthorized"}`,
			want:   true,
		},
		{
			name:   "401 empty body",
			status: 401,
			body:   "",
			want:   true,
		},
		// 403 with auth signals.
		{
			name:   "403 with csrf in body",
			status: 403,
			body:   "<html>CSRF token mismatch</html>",
			want:   true,
		},
		{
			name:   "403 with session in body",
			status: 403,
			body:   "<html>Session expired</html>",
			want:   true,
		},
		{
			name:   "403 with authenticity in body",
			status: 403,
			body:   "<html>Invalid authenticity token</html>",
			want:   true,
		},
		{
			name:   "403 without auth signal",
			status: 403,
			body:   `{"message":"Forbidden"}`,
			want:   false,
		},
		// 302/303 redirects.
		{
			name:    "302 to login",
			status:  302,
			headers: map[string]string{"Location": "https://school.instructure.com/login"},
			baseURL: "https://school.instructure.com",
			want:    true,
		},
		{
			name:    "303 to logout",
			status:  303,
			headers: map[string]string{"Location": "https://school.instructure.com/logout"},
			baseURL: "https://school.instructure.com",
			want:    true,
		},
		{
			name:    "302 to saml",
			status:  302,
			headers: map[string]string{"Location": "https://school.instructure.com/saml/sso"},
			baseURL: "https://school.instructure.com",
			want:    true,
		},
		{
			name:    "302 to shibboleth host",
			status:  302,
			headers: map[string]string{"Location": "https://shibboleth.school.edu/sso"},
			baseURL: "https://school.instructure.com",
			want:    true,
		},
		{
			name:    "302 to cas host",
			status:  302,
			headers: map[string]string{"Location": "https://cas.school.edu/login"},
			baseURL: "https://school.instructure.com",
			want:    true,
		},
		{
			name:    "302 external host with auth prefix",
			status:  302,
			headers: map[string]string{"Location": "https://idp.school.edu/sso/saml"},
			baseURL: "https://school.instructure.com",
			want:    true,
		},
		{
			name:    "302 same host non-auth path",
			status:  302,
			headers: map[string]string{"Location": "https://school.instructure.com/dashboard"},
			baseURL: "https://school.instructure.com",
			want:    false,
		},
		{
			name:    "302 external host non-auth path",
			status:  302,
			headers: map[string]string{"Location": "https://other.example.com/dashboard"},
			baseURL: "https://school.instructure.com",
			want:    false,
		},
		{
			name:    "302 missing location header",
			status:  302,
			baseURL: "https://school.instructure.com",
			want:    false,
		},
		// 200 with HTML login page.
		{
			name:       "200 html login page with json accept",
			status:     200,
			body:       "<html><login page></html>",
			headers:    map[string]string{"Content-Type": "text/html"},
			baseURL:    "https://school.instructure.com",
			reqHeaders: map[string]string{"Accept": "application/json"},
			want:       true,
		},
		{
			name:       "200 html with canvas json accept",
			status:     200,
			body:       "<html><login page></html>",
			headers:    map[string]string{"Content-Type": "text/html"},
			baseURL:    "https://school.instructure.com",
			reqHeaders: map[string]string{"Accept": "application/json+canvas-string-ids"},
			want:       true,
		},
		{
			name:       "200 html without json accept",
			status:     200,
			body:       "<html>Some page</html>",
			headers:    map[string]string{"Content-Type": "text/html"},
			baseURL:    "https://school.instructure.com",
			reqHeaders: map[string]string{"Accept": "text/html"},
			want:       false,
		},
		{
			name:    "200 html no request object",
			status:  200,
			body:    "<html>Some page</html>",
			headers: map[string]string{"Content-Type": "text/html"},
			baseURL: "https://school.instructure.com",
			want:    false,
		},
		// 200 JSON success.
		{
			name:    "200 json success",
			status:  200,
			body:    `{"id":1,"name":"Course"}`,
			headers: map[string]string{"Content-Type": "application/json"},
			baseURL: "https://school.instructure.com",
			want:    false,
		},
		{
			name:    "200 canvas json success",
			status:  200,
			body:    `{"id":"1","name":"Course"}`,
			headers: map[string]string{"Content-Type": "application/json+canvas-string-ids"},
			baseURL: "https://school.instructure.com",
			want:    false,
		},
		// 404 is not auth failure.
		{
			name:   "404 not found",
			status: 404,
			body:   `{"message":"Not Found"}`,
			want:   false,
		},
		// Body CSRF error (scoped to 422).
		{
			name:   "422 with authenticity token in body",
			status: 422,
			body:   `{"message":"Invalid authenticity token"}`,
			want:   true,
		},
		{
			name:    "200 text with authenticity token in body",
			status:  200,
			body:    "<html> authenticity token mismatch</html>",
			headers: map[string]string{"Content-Type": "text/plain"},
			want:    false,
		},
		{
			name:   "500 with csrf in body",
			status: 500,
			body:   "<html>CSRF token invalid</html>",
			want:   false,
		},
		// Normal server error without CSRF.
		{
			name:   "500 server error without csrf",
			status: 500,
			body:   `{"message":"Internal Server Error"}`,
			want:   false,
		},
		// Normal 200 with text/plain (no CSRF in body).
		{
			name:    "200 text plain no csrf",
			status:  200,
			body:    "OK",
			headers: map[string]string{"Content-Type": "text/plain"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			for k, v := range tt.headers {
				h.Set(k, v)
			}
			resp := &http.Response{
				StatusCode: tt.status,
				Header:     h,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			if tt.reqHeaders != nil {
				req, _ := http.NewRequest("GET", "https://example.com", nil)
				for k, v := range tt.reqHeaders {
					req.Header.Set(k, v)
				}
				resp.Request = req
			}
			got := IsCookieSessionExpired(resp, []byte(tt.body), tt.baseURL)
			if got != tt.want {
				t.Errorf("IsCookieSessionExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeError_EmptyBody(t *testing.T) {
	resp := newResponse(400, "", nil)
	env := NormalizeError(resp, "api.get")

	if env.Error.Message != "Bad Request" {
		t.Errorf("Message = %q, want %q (fallback to StatusText)", env.Error.Message, "Bad Request")
	}
	if env.Error.Code != "CANVAS_API_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_API_ERROR")
	}
}

func TestNormalizeError_NoMessageField(t *testing.T) {
	resp := newResponse(400, `{"errors":[{"message":"some error"}]}`, nil)
	env := NormalizeError(resp, "api.get")

	// Body has no top-level "message" field, so should fall back to StatusText.
	if env.Error.Message != "Bad Request" {
		t.Errorf("Message = %q, want %q (fallback to StatusText)", env.Error.Message, "Bad Request")
	}
}

func TestNormalizeError_SessionExpired(t *testing.T) {
	// 401 with cookie auth baseURL triggers session expired detection.
	resp := newResponse(401, `{"message":"Unauthorized"}`, nil)
	env := NormalizeError(resp, "courses.list", "https://school.instructure.com")

	if env.Error.Code != "CANVAS_SESSION_EXPIRED" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_SESSION_EXPIRED")
	}
	if env.Error.Category != "auth" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "auth")
	}
	if !strings.Contains(env.Error.Message, "session expired") {
		t.Errorf("Message = %q, want it to contain 'session expired'", env.Error.Message)
	}
}

func TestNormalizeError_SessionExpired_EmptyBaseURL(t *testing.T) {
	// Empty baseURL should NOT trigger session expired.
	resp := newResponse(401, `{"message":"Unauthorized"}`, nil)
	env := NormalizeError(resp, "courses.list", "")

	if env.Error.Code == "CANVAS_SESSION_EXPIRED" {
		t.Error("should not detect session expired with empty baseURL")
	}
}

func TestNormalizeError_SessionExpired_NoBaseURL(t *testing.T) {
	// No baseURL variadic arg should NOT trigger session expired.
	resp := newResponse(401, `{"message":"Unauthorized"}`, nil)
	env := NormalizeError(resp, "courses.list")

	if env.Error.Code == "CANVAS_SESSION_EXPIRED" {
		t.Error("should not detect session expired without baseURL")
	}
}

func TestNormalizeErrorFromBody_EmptyBody(t *testing.T) {
	resp := newResponse(500, "", nil)
	errInfo := NormalizeErrorFromBody(resp, []byte(""))

	if errInfo.Message != "Internal Server Error" {
		t.Errorf("Message = %q, want %q", errInfo.Message, "Internal Server Error")
	}
	if errInfo.Code != "CANVAS_SERVER_ERROR" {
		t.Errorf("Code = %q, want %q", errInfo.Code, "CANVAS_SERVER_ERROR")
	}
}

func TestHasAuthPathPrefix_AllPrefixes(t *testing.T) {
	tests := []struct {
		location string
		want     bool
	}{
		{"https://school.edu/login", true},
		{"https://school.edu/auth/sso", true},
		{"https://school.edu/sso/saml", true},
		{"https://school.edu/cas/login", true},
		{"https://school.edu/saml/sso", true},
		{"https://school.edu/idp/SSO", true},
		{"https://school.edu/shibboleth/sso", true},
		{"https://school.edu/signin", true},
		{"https://school.edu/sign-in", true},
		{"https://school.edu/Login", true}, // case insensitive
		{"https://school.edu/api/v1", false},
		{"https://school.edu/dashboard", false},
	}
	for _, tt := range tests {
		t.Run(tt.location, func(t *testing.T) {
			got := hasAuthPathPrefix(tt.location)
			if got != tt.want {
				t.Errorf("hasAuthPathPrefix(%q) = %v, want %v", tt.location, got, tt.want)
			}
		})
	}
}

func TestIsAuthRedirect_HostPatterns(t *testing.T) {
	tests := []struct {
		location string
		want     bool
	}{
		{"https://shibboleth.university.edu/sso", true},
		{"https://idp.shibboleth.university.edu/sso", true},
		{"https://cas.university.edu/login", true},
		{"https://auth.cas.university.edu/login", true},
		{"https://regular.school.edu/dashboard", false},
	}
	for _, tt := range tests {
		t.Run(tt.location, func(t *testing.T) {
			got := isAuthRedirect(tt.location)
			if got != tt.want {
				t.Errorf("isAuthRedirect(%q) = %v, want %v", tt.location, got, tt.want)
			}
		})
	}
}

func TestHostMatches(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		location string
		want     bool
	}{
		{"same host", "https://school.instructure.com", "https://school.instructure.com/login", true},
		{"different host", "https://school.instructure.com", "https://idp.school.edu/sso", false},
		{"same host with port", "https://school.instructure.com:8080", "https://school.instructure.com:8080/api", true},
		{"different port", "https://school.instructure.com:8080", "https://school.instructure.com:9090/api", false},
		{"case insensitive", "https://School.Instructure.Com", "https://school.instructure.com/login", true},
		{"empty base", "", "https://school.instructure.com/login", false},
		{"empty location", "https://school.instructure.com", "", false},
		{"both empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hostMatches(tt.baseURL, tt.location)
			if got != tt.want {
				t.Errorf("hostMatches(%q, %q) = %v, want %v", tt.baseURL, tt.location, got, tt.want)
			}
		})
	}
}
