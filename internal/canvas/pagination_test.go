package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseLinkHeaderNextAndPrev(t *testing.T) {
	header := `<https://canvas.example.com/api/v1/courses?page=2>; rel="next", <https://canvas.example.com/api/v1/courses?page=1>; rel="prev"`
	links := ParseLinkHeader(header)

	if links["next"] != "https://canvas.example.com/api/v1/courses?page=2" {
		t.Errorf("next = %q, want %q", links["next"], "https://canvas.example.com/api/v1/courses?page=2")
	}
	if links["prev"] != "https://canvas.example.com/api/v1/courses?page=1" {
		t.Errorf("prev = %q, want %q", links["prev"], "https://canvas.example.com/api/v1/courses?page=1")
	}
}

func TestParseLinkHeaderCaseInsensitive(t *testing.T) {
	header := `<https://canvas.example.com/api/v1/courses?page=3>; rel="Next"`
	links := ParseLinkHeader(header)

	if links["next"] != "https://canvas.example.com/api/v1/courses?page=3" {
		t.Errorf("next = %q, want %q", links["next"], "https://canvas.example.com/api/v1/courses?page=3")
	}
}

func TestParseLinkHeaderEmpty(t *testing.T) {
	links := ParseLinkHeader("")
	if len(links) != 0 {
		t.Errorf("expected empty map, got %d entries", len(links))
	}
}

func TestParseLinkHeaderLast(t *testing.T) {
	header := `<https://canvas.example.com/api/v1/courses?page=5>; rel="last"`
	links := ParseLinkHeader(header)
	if links["last"] != "https://canvas.example.com/api/v1/courses?page=5" {
		t.Errorf("last = %q, want %q", links["last"], "https://canvas.example.com/api/v1/courses?page=5")
	}
}

func TestPaginateFollowsNextUntilExhausted(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	page := 0
	var srv *httptest.Server //nolint:staticcheck // self-referential closure
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		var items []item
		switch page {
		case 1:
			items = []item{{ID: "1"}, {ID: "2"}}
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/items?page=2>; rel="next"`, srv.URL))
		case 2:
			items = []item{{ID: "3"}}
			// No Link header = last page
		default:
			items = []item{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 0, 100)
	if err != nil {
		t.Fatalf("Paginate() error: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("len(items) = %d, want 3", len(items))
	}
	if meta.RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2", meta.RequestCount)
	}
	if meta.TotalItems != 3 {
		t.Errorf("TotalItems = %d, want 3", meta.TotalItems)
	}
	if !meta.Paginated {
		t.Error("Paginated should be true")
	}
}

func TestPaginateRespectsLimit(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []item{{ID: "1"}, {ID: "2"}, {ID: "3"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 2, 100)
	if err != nil {
		t.Fatalf("Paginate() error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if meta.TotalItems != 2 {
		t.Errorf("TotalItems = %d, want 2", meta.TotalItems)
	}
	if meta.Limit != 2 {
		t.Errorf("Limit = %d, want 2", meta.Limit)
	}
}

func TestPaginateEmptyFirstPage(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]item{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 0, 100)
	if err != nil {
		t.Fatalf("Paginate() error: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
	if meta.RequestCount != 1 {
		t.Errorf("RequestCount = %d, want 1", meta.RequestCount)
	}
	if meta.TotalItems != 0 {
		t.Errorf("TotalItems = %d, want 0", meta.TotalItems)
	}
}

func TestPaginateMissingLinkHeaderIsSinglePage(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		items := []item{{ID: "a"}, {ID: "b"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 0, 100)
	if err != nil {
		t.Fatalf("Paginate() error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if reqCount != 1 {
		t.Errorf("request count = %d, want 1", reqCount)
	}
	if meta.RequestCount != 1 {
		t.Errorf("RequestCount = %d, want 1", meta.RequestCount)
	}
}

func TestPaginatePassesPageSizeQuery(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		items := []item{{ID: "1"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, _, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 0, 50)
	if err != nil {
		t.Fatalf("Paginate() error: %v", err)
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "50" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "50")
	}
}

func TestPaginate_APIErrorResponse(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"invalid parameter"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, _, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 0, 100)
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "api error") {
		t.Errorf("error = %q, want it to contain 'api error'", err.Error())
	}
}

func TestPaginate_APIErrorWithCookieAuth(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "0.1.0", 5*time.Second, 0).WithCookie("cookie", "")
	_, meta, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 0, 100)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if meta.RequestCount != 1 {
		t.Errorf("RequestCount = %d, want 1", meta.RequestCount)
	}
}

func TestPaginate_LimitTruncatesPageItems(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []item{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 3, 100)
	if err != nil {
		t.Fatalf("Paginate() error: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("len(items) = %d, want 3", len(items))
	}
	if meta.TotalItems != 3 {
		t.Errorf("TotalItems = %d, want 3", meta.TotalItems)
	}
}

func TestPaginate_RelativeNextURL(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	page := 0
	var srv *httptest.Server //nolint:staticcheck // self-referential closure
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		var items []item
		switch page {
		case 1:
			items = []item{{ID: "1"}, {ID: "2"}}
			// Use a relative path (not an absolute URL) in the Link header.
			w.Header().Set("Link", `</api/v1/items?page=2>; rel="next"`)
		case 2:
			items = []item{{ID: "3"}}
			// No Link header = last page
		default:
			items = []item{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := Paginate[item](context.Background(), c, "/api/v1/items", nil, 0, 100)
	if err != nil {
		t.Fatalf("Paginate() error: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("len(items) = %d, want 3", len(items))
	}
	if meta.RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestParseLinkHeader_MissingAngleBrackets(t *testing.T) {
	header := `https://canvas.example.com/api/v1/courses?page=2; rel="next"`
	links := ParseLinkHeader(header)
	if len(links) != 0 {
		t.Errorf("expected empty map for malformed link, got %d entries", len(links))
	}
}

func TestParseLinkHeader_NoRelAttribute(t *testing.T) {
	header := `<https://canvas.example.com/api/v1/courses?page=2>; title="next"`
	links := ParseLinkHeader(header)
	if len(links) != 0 {
		t.Errorf("expected empty map for link without rel, got %d entries", len(links))
	}
}

func TestPaginateContextCancellation(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	var srv *httptest.Server //nolint:staticcheck // self-referential closure
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []item{{ID: "1"}}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/items?page=2>; rel="next"`, srv.URL))
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err := Paginate[item](ctx, c, "/api/v1/items", nil, 0, 100)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
