package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestDefaultRoutes(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantKey    string // a key expected in the JSON response
	}{
		{
			name:       "GET /api/v1/users/self returns default user",
			method:     "GET",
			path:       "/api/v1/users/self",
			wantStatus: http.StatusOK,
			wantKey:    "Test User",
		},
		{
			name:       "GET /api/v1/courses returns default course list",
			method:     "GET",
			path:       "/api/v1/courses",
			wantStatus: http.StatusOK,
			wantKey:    "Test Course",
		},
		{
			name:       "GET /api/v1/courses/1 returns default course",
			method:     "GET",
			path:       "/api/v1/courses/1",
			wantStatus: http.StatusOK,
			wantKey:    "TC101",
		},
		{
			name:       "unregistered route returns 404",
			method:     "GET",
			path:       "/api/v1/nope",
			wantStatus: http.StatusNotFound,
			wantKey:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(m.URL() + tt.path)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			body, _ := io.ReadAll(resp.Body)
			if len(tt.wantKey) > 0 && !json.Valid(body) {
				t.Errorf("response is not valid JSON: %s", body)
			}
			if len(tt.wantKey) > 0 && !contains(string(body), tt.wantKey) {
				t.Errorf("response body does not contain %q: %s", tt.wantKey, body)
			}
		})
	}
}

func TestRequestLogging(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	// Make two requests.
	http.Get(m.URL() + "/api/v1/users/self")
	http.Get(m.URL() + "/api/v1/courses")

	if m.RequestCount() != 2 {
		t.Fatalf("RequestCount() = %d, want 2", m.RequestCount())
	}

	// Check first request.
	req0 := m.LastRequest()
	if req0 == nil {
		t.Fatal("LastRequest() returned nil after requests")
	}

	// LastRequest should be the courses request.
	if req0.Path != "/api/v1/courses" {
		t.Errorf("LastRequest().Path = %q, want /api/v1/courses", req0.Path)
	}
	if req0.Method != "GET" {
		t.Errorf("LastRequest().Method = %q, want GET", req0.Method)
	}
}

func TestRequestLoggingHeaders(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	req, _ := http.NewRequest("GET", m.URL()+"/api/v1/users/self", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	req.Header.Set("Accept", "application/json+canvas-string-ids")
	http.DefaultClient.Do(req)

	if m.RequestCount() != 1 {
		t.Fatalf("RequestCount() = %d, want 1", m.RequestCount())
	}

	last := m.LastRequest()
	if last == nil {
		t.Fatal("LastRequest() returned nil")
	}

	if got := last.Headers.Get("Authorization"); got != "Bearer test-token-123" {
		t.Errorf("Authorization header = %q, want Bearer test-token-123", got)
	}
	if got := last.Headers.Get("Accept"); got != "application/json+canvas-string-ids" {
		t.Errorf("Accept header = %q, want application/json+canvas-string-ids", got)
	}
}

func TestPaginationSimulation(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	page1 := []map[string]any{
		{"id": "1", "name": "Item 1"},
		{"id": "2", "name": "Item 2"},
	}
	page2 := []map[string]any{
		{"id": "3", "name": "Item 3"},
	}

	m.SetPagination("/api/v1/courses/1/assignments", [][]map[string]any{page1, page2})

	// First page should have a Link header.
	resp1, err := http.Get(m.URL() + "/api/v1/courses/1/assignments")
	if err != nil {
		t.Fatalf("page 1 request failed: %v", err)
	}
	defer resp1.Body.Close()

	linkHeader := resp1.Header.Get("Link")
	if linkHeader == "" {
		t.Fatal("page 1 response missing Link header")
	}
	if !contains(linkHeader, `rel="next"`) {
		t.Errorf("Link header missing rel=next: %s", linkHeader)
	}

	body1, _ := io.ReadAll(resp1.Body)
	var items1 []map[string]any
	if err := json.Unmarshal(body1, &items1); err != nil {
		t.Fatalf("page 1 JSON decode failed: %v", err)
	}
	if len(items1) != 2 {
		t.Errorf("page 1 has %d items, want 2", len(items1))
	}

	// Second (last) page should NOT have a Link header.
	resp2, err := http.Get(m.URL() + "/api/v1/courses/1/assignments")
	if err != nil {
		t.Fatalf("page 2 request failed: %v", err)
	}
	defer resp2.Body.Close()

	if link := resp2.Header.Get("Link"); link != "" {
		t.Errorf("last page should not have Link header, got %q", link)
	}

	body2, _ := io.ReadAll(resp2.Body)
	var items2 []map[string]any
	if err := json.Unmarshal(body2, &items2); err != nil {
		t.Fatalf("page 2 JSON decode failed: %v", err)
	}
	if len(items2) != 1 {
		t.Errorf("page 2 has %d items, want 1", len(items2))
	}
}

func TestRateLimitHeaders(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	m.SetRateLimitHeaders(1.5, 48.5)

	resp, err := http.Get(m.URL() + "/api/v1/users/self")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Request-Cost"); got != "1.5" {
		t.Errorf("X-Request-Cost = %q, want 1.5", got)
	}
	if got := resp.Header.Get("X-Rate-Limit-Remaining"); got != "48.5" {
		t.Errorf("X-Rate-Limit-Remaining = %q, want 48.5", got)
	}
}

func TestSetRetryAfter(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	m.SetRetryAfter(30)

	resp, err := http.Get(m.URL() + "/api/v1/users/self")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}
	if got := resp.Header.Get("Retry-After"); got != "30" {
		t.Errorf("Retry-After = %q, want 30", got)
	}

	// Disable retry-after and verify normal response resumes.
	m.SetRetryAfter(0)

	resp2, err := http.Get(m.URL() + "/api/v1/users/self")
	if err != nil {
		t.Fatalf("request after disable failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("after disabling retry-after: status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}
}

func TestOnCustomRoute(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	m.On("POST", "/api/v1/courses/1/assignments", http.StatusCreated, map[string]any{
		"id":   "42",
		"name": "Homework 1",
	})

	resp, err := http.Post(m.URL()+"/api/v1/courses/1/assignments", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}
	if result["id"] != "42" {
		t.Errorf("id = %v, want 42", result["id"])
	}
	if result["name"] != "Homework 1" {
		t.Errorf("name = %v, want Homework 1", result["name"])
	}
}

func TestReset(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	http.Get(m.URL() + "/api/v1/users/self")
	http.Get(m.URL() + "/api/v1/courses")

	if m.RequestCount() != 2 {
		t.Fatalf("before reset: RequestCount() = %d, want 2", m.RequestCount())
	}

	m.Reset()

	if m.RequestCount() != 0 {
		t.Errorf("after reset: RequestCount() = %d, want 0", m.RequestCount())
	}
	if m.LastRequest() != nil {
		t.Error("after reset: LastRequest() should be nil")
	}
}

func TestClose(t *testing.T) {
	m := NewMockCanvas()
	serverURL := m.URL()

	// Verify server is working.
	resp, err := http.Get(serverURL + "/api/v1/users/self")
	if err != nil {
		t.Fatalf("request before close failed: %v", err)
	}
	resp.Body.Close()

	m.Close()

	// Server should be closed now.
	_, err = http.Get(serverURL + "/api/v1/users/self")
	if err == nil {
		t.Error("expected error after Close(), got nil")
	}
}

func TestURL(t *testing.T) {
	m := NewMockCanvas()
	defer m.Close()

	u := m.URL()
	if u == "" {
		t.Error("URL() returned empty string")
	}
	if !contains(u, "http") {
		t.Errorf("URL() = %q, does not start with http", u)
	}
}

// contains is a simple substring check helper.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
