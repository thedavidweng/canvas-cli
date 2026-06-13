// Package testutil provides shared test helpers for canvas-cli.
package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
)

// RecordedRequest captures the details of a single HTTP request
// received by the mock server for assertion in tests.
type RecordedRequest struct {
	Method  string
	Path    string
	Query   url.Values
	Headers http.Header
	Body    string
}

// route holds the handler configuration for a registered path.
type route struct {
	statusCode int
	body       any
}

// paginationConfig stores per-path pagination state.
type paginationConfig struct {
	pages     [][]map[string]any
	nextIndex int
}

// MockCanvas is a test HTTP server that simulates Canvas API endpoints.
// It records incoming requests and allows tests to configure responses
// for specific routes, including paginated and rate-limited responses.
type MockCanvas struct {
	Server     *httptest.Server
	routes     map[string]map[string]route // method -> path -> route
	requestLog []RecordedRequest
	mu         sync.Mutex

	// rateLimitCost is the value returned in X-Request-Cost header.
	rateLimitCost float64
	// rateLimitRemaining is the value returned in X-Rate-Limit-Remaining header.
	rateLimitRemaining float64
	// hasRateLimit controls whether rate limit headers are included.
	hasRateLimit bool

	// retryAfterSeconds, when > 0, causes all requests to return 429 with Retry-After.
	retryAfterSeconds int

	// pagination holds per-path pagination configurations.
	pagination map[string]*paginationConfig

	// redirects stores path -> redirectURL for routes that return 302.
	redirects map[string]string
}

// NewMockCanvas creates and starts a new mock Canvas HTTP server.
// It registers default routes for common endpoints:
//   - GET /api/v1/users/self
//   - GET /api/v1/courses
//   - GET /api/v1/courses/1
func NewMockCanvas() *MockCanvas {
	m := &MockCanvas{
		routes:     make(map[string]map[string]route),
		pagination: make(map[string]*paginationConfig),
		redirects:  make(map[string]string),
	}

	m.Server = httptest.NewServer(http.HandlerFunc(m.handler))

	// Register default routes.
	m.On("GET", "/api/v1/users/self", http.StatusOK, map[string]any{
		"id":            "1",
		"name":          "Test User",
		"sortable_name": "User, Test",
		"short_name":    "Test",
	})

	m.On("GET", "/api/v1/courses", http.StatusOK, []map[string]any{
		{
			"id":             "1",
			"name":           "Test Course",
			"course_code":    "TC101",
			"workflow_state": "available",
		},
	})

	m.On("GET", "/api/v1/courses/1", http.StatusOK, map[string]any{
		"id":          "1",
		"name":        "Test Course",
		"course_code": "TC101",
	})

	return m
}

// handler is the central HTTP handler that dispatches to registered routes
// and records each request for later assertion.
func (m *MockCanvas) handler(w http.ResponseWriter, r *http.Request) {
	// Record the request.
	bodyBytes, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	m.requestLog = append(m.requestLog, RecordedRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.Query(),
		Headers: r.Header.Clone(),
		Body:    string(bodyBytes),
	})
	m.mu.Unlock()

	// If retry-after is configured, always return 429.
	m.mu.Lock()
	retryAfter := m.retryAfterSeconds
	m.mu.Unlock()
	if retryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintf(w, `{"errors":[{"message":"rate limit exceeded"}]}`)
		return
	}

	// Set rate limit headers if configured.
	m.mu.Lock()
	hasRL := m.hasRateLimit
	rlCost := m.rateLimitCost
	rlRem := m.rateLimitRemaining
	m.mu.Unlock()

	if hasRL {
		w.Header().Set("X-Request-Cost", fmt.Sprintf("%g", rlCost))
		w.Header().Set("X-Rate-Limit-Remaining", fmt.Sprintf("%g", rlRem))
	}

	// Check for pagination configuration.
	m.mu.Lock()
	pgCfg, hasPagination := m.pagination[r.URL.Path]
	m.mu.Unlock()

	if hasPagination && pgCfg.nextIndex < len(pgCfg.pages) {
		m.mu.Lock()
		pageData := pgCfg.pages[pgCfg.nextIndex]
		isLast := pgCfg.nextIndex == len(pgCfg.pages)-1
		pgCfg.nextIndex++
		m.mu.Unlock()

		// Set Link header for non-last pages.
		if !isLast {
			nextURL := fmt.Sprintf("%s%s?page=%d", m.Server.URL, r.URL.Path, pgCfg.nextIndex+1)
			w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, nextURL))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(pageData)
		return
	}

	// Check for redirect configuration.
	m.mu.Lock()
	redirectURL, hasRedirect := m.redirects[r.Method+":"+r.URL.Path]
	m.mu.Unlock()

	if hasRedirect {
		w.Header().Set("Location", m.Server.URL+redirectURL)
		w.WriteHeader(http.StatusFound)
		return
	}

	// Look up registered route.
	m.mu.Lock()
	methodRoutes, ok := m.routes[r.Method]
	var rt route
	if ok {
		rt, ok = methodRoutes[r.URL.Path]
	}
	m.mu.Unlock()

	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"errors":[{"message":"not found"}]}`)
		return
	}

	w.WriteHeader(rt.statusCode)
	switch b := rt.body.(type) {
	case []byte:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(b)
	case string:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte(b))
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(b)
	}
}

// On registers a handler for the given HTTP method and path.
// When the mock receives a matching request it will respond with
// the given status code and JSON-serialized body.
func (m *MockCanvas) On(method, path string, statusCode int, body any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	method = strings.ToUpper(method)
	if m.routes[method] == nil {
		m.routes[method] = make(map[string]route)
	}
	m.routes[method][path] = route{
		statusCode: statusCode,
		body:       body,
	}
}

// SetPagination configures paginated responses for the given path.
// Each entry in pages is one page of results. The mock will serve
// pages sequentially, including Link headers with rel="next" for
// all pages except the last.
func (m *MockCanvas) SetPagination(path string, pages [][]map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pagination[path] = &paginationConfig{
		pages:     pages,
		nextIndex: 0,
	}
}

// SetRateLimitHeaders configures the mock to include rate limit
// response headers (X-Request-Cost and X-Rate-Limit-Remaining)
// on every response.
func (m *MockCanvas) SetRateLimitHeaders(cost, remaining float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hasRateLimit = true
	m.rateLimitCost = cost
	m.rateLimitRemaining = remaining
}

// SetRetryAfter configures the mock to return HTTP 429 with a
// Retry-After header on every request. Pass 0 to disable.
func (m *MockCanvas) SetRetryAfter(seconds int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.retryAfterSeconds = seconds
}

// LastRequest returns the most recent RecordedRequest, or nil if
// no requests have been received.
func (m *MockCanvas) LastRequest() *RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.requestLog) == 0 {
		return nil
	}
	rr := m.requestLog[len(m.requestLog)-1]
	return &rr
}

// RequestCount returns the number of requests received by the mock server.
func (m *MockCanvas) RequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.requestLog)
}

// Reset clears the request log and resets pagination state.
func (m *MockCanvas) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestLog = nil
	for _, cfg := range m.pagination {
		cfg.nextIndex = 0
	}
}

// OnUploadRedirect registers a route that responds with HTTP 302 and a
// Location header pointing to redirectPath. This is used to simulate
// Canvas file upload initiation which redirects to the actual upload URL.
func (m *MockCanvas) OnUploadRedirect(method, path, redirectPath string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.redirects[strings.ToUpper(method)+":"+path] = redirectPath
}

// OnUploadInit registers a route that returns a 200 with upload_url and
// upload_params, simulating step 1 of the Canvas 3-step file upload flow.
// uploadPath is the path the client should POST the file content to.
func (m *MockCanvas) OnUploadInit(method, path, uploadPath string, params map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullUploadURL := m.Server.URL + uploadPath
	methodKey := strings.ToUpper(method)
	if m.routes[methodKey] == nil {
		m.routes[methodKey] = make(map[string]route)
	}
	m.routes[methodKey][path] = route{
		statusCode: http.StatusOK,
		body: map[string]any{
			"upload_url":    fullUploadURL,
			"upload_params": params,
		},
	}
}

// RequestLog returns a copy of all recorded requests.
func (m *MockCanvas) RequestLog() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	log := make([]RecordedRequest, len(m.requestLog))
	copy(log, m.requestLog)
	return log
}

// Close shuts down the underlying httptest.Server.
func (m *MockCanvas) Close() {
	m.Server.Close()
}

// URL returns the base URL of the mock server.
func (m *MockCanvas) URL() string {
	return m.Server.URL
}
