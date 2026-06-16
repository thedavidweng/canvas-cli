package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestDecodesJSONResponse(t *testing.T) {
	type course struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Cost", "0.8")
		w.Header().Set("X-Rate-Limit-Remaining", "999.2")
		json.NewEncoder(w).Encode(course{ID: "123", Name: "Test Course"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var result course
	meta, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/courses/123",
		DecodeInto: &result,
	})
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}

	if result.ID != "123" {
		t.Errorf("ID = %q, want %q", result.ID, "123")
	}
	if result.Name != "Test Course" {
		t.Errorf("Name = %q, want %q", result.Name, "Test Course")
	}
	if meta.RateLimit == nil {
		t.Fatal("RateLimit should not be nil")
	}
	if meta.RateLimit.RequestCost != 0.8 {
		t.Errorf("RequestCost = %f, want 0.8", meta.RateLimit.RequestCost)
	}
	if meta.RateLimit.Remaining != 999.2 {
		t.Errorf("Remaining = %f, want 999.2", meta.RateLimit.Remaining)
	}
}

func TestRequestHandlesPagination(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			items := []item{{ID: "1"}, {ID: "2"}}
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/items?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode(items)
		case 2:
			items := []item{{ID: "3"}}
			json.NewEncoder(w).Encode(items)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var items []item
	meta, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/items",
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &items,
	})
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("len(items) = %d, want 3", len(items))
	}
	if !meta.Pagination.Paginated {
		t.Error("Pagination.Paginated should be true")
	}
	if meta.Pagination.RequestCount != 2 {
		t.Errorf("Pagination.RequestCount = %d, want 2", meta.Pagination.RequestCount)
	}
	if meta.Pagination.TotalItems != 3 {
		t.Errorf("Pagination.TotalItems = %d, want 3", meta.Pagination.TotalItems)
	}
}

func TestRequestCapturesRateLimitMeta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Cost", "2.0")
		w.Header().Set("X-Rate-Limit-Remaining", "995.0")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var result map[string]string
	meta, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/test",
		DecodeInto: &result,
	})
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}

	if meta.RateLimit == nil {
		t.Fatal("RateLimit should not be nil")
	}
	if meta.RateLimit.RequestCost != 2.0 {
		t.Errorf("RequestCost = %f, want 2.0", meta.RateLimit.RequestCost)
	}
	if meta.RateLimit.Remaining != 995.0 {
		t.Errorf("Remaining = %f, want 995.0", meta.RateLimit.Remaining)
	}
}

func TestRequestNoPaginationWhenPaginateFalse(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Set a Link header that would normally trigger pagination
		w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/items?page=2>; rel="next"`, srv.URL))
		json.NewEncoder(w).Encode([]item{{ID: "1"}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var items []item
	meta, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/items",
		Paginate:   false,
		DecodeInto: &items,
	})
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
	if meta.Pagination.Paginated {
		t.Error("Pagination.Paginated should be false when Paginate option is false")
	}
}

func TestRequestNoDecodeInto(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	meta, err := Request(context.Background(), c, RequestOptions{
		Method:    "GET",
		PathOrURL: "/api/v1/test",
	})
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
	if meta == nil {
		t.Fatal("meta should not be nil")
	}
}

func TestRequest_PaginateWithoutDecodeInto(t *testing.T) {
	c := NewClient("https://example.com", "tok", "0.1.0", 5*time.Second, 0)
	_, err := Request(context.Background(), c, RequestOptions{
		Method:    "GET",
		PathOrURL: "/api/v1/items",
		Paginate:  true,
		PageSize:  100,
		// DecodeInto is nil
	})
	if err == nil {
		t.Fatal("expected error when Paginate=true and DecodeInto=nil")
	}
	if !strings.Contains(err.Error(), "decodeInto required") {
		t.Errorf("error = %q, want it to contain 'decodeInto required'", err.Error())
	}
}

func TestRequest_PaginateDefaultPageSize(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]item{{ID: "1"}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var items []item
	_, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/items",
		Paginate:   true,
		PageSize:   0, // should default to 100
		DecodeInto: &items,
	})
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
}

func TestRequest_APIErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var result map[string]string
	_, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/courses/999",
		DecodeInto: &result,
	})
	if err == nil {
		t.Fatal("expected error for 404 status")
	}
	if !strings.Contains(err.Error(), "api error") {
		t.Errorf("error = %q, want it to contain 'api error'", err.Error())
	}
}

func TestRequest_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var result map[string]string
	_, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/test",
		DecodeInto: &result,
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("error = %q, want it to contain 'failed to decode response'", err.Error())
	}
}

func TestRequestPassesQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	var result map[string]string
	_, err := Request(context.Background(), c, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/test",
		Query:      map[string][]string{"search": {"test"}},
		DecodeInto: &result,
	})
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}

	if gotQuery != "search=test" {
		t.Errorf("query = %q, want %q", gotQuery, "search=test")
	}
}
