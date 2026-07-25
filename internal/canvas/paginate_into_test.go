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

func TestPaginateInto_InvalidDst(t *testing.T) {
	c := NewClient("https://example.com", "tok", "0.1.0", 5*time.Second, 0)

	// dst is not a pointer to slice.
	_, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 0, 100, "not-a-slice")
	if err == nil {
		t.Fatal("expected error for non-slice dst")
	}
	if !strings.Contains(err.Error(), "must be a pointer to slice") {
		t.Errorf("error = %q, want it to contain 'must be a pointer to slice'", err.Error())
	}
}

func TestPaginateInto_LimitStopsEarly(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]item{
			{ID: "1"}, {ID: "2"}, {ID: "3"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var items []item
	meta, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 2, 100, &items)
	if err != nil {
		t.Fatalf("paginateInto() error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "1" || items[1].ID != "2" {
		t.Errorf("items = %v, want [1, 2]", items)
	}
	if meta.TotalItems != 2 {
		t.Errorf("TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestPaginateInto_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var items []map[string]string
	_, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 0, 100, &items)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "api error") {
		t.Errorf("error = %q, want it to contain 'api error'", err.Error())
	}
}

func TestPaginateInto_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var items []map[string]string
	_, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 0, 100, &items)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode page") {
		t.Errorf("error = %q, want it to contain 'failed to decode page'", err.Error())
	}
}

func TestPaginateInto_ItemDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Array with an element that can't decode into a string field.
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 123},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	type item struct {
		ID string `json:"id"`
	}
	var items []item
	_, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 0, 100, &items)
	if err == nil {
		t.Fatal("expected error for item decode failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode item") {
		t.Errorf("error = %q, want it to contain 'failed to decode item'", err.Error())
	}
}

func TestPaginateInto_RelativeNextURL(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			// Relative next URL (no scheme/host).
			w.Header().Set("Link", `</api/v1/items?page=2>; rel="next"`)
			_ = json.NewEncoder(w).Encode([]item{{ID: "1"}})
		case 2:
			_ = json.NewEncoder(w).Encode([]item{{ID: "2"}})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var items []item
	meta, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 0, 100, &items)
	if err != nil {
		t.Fatalf("paginateInto() error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if meta.RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestPaginateInto_AbsoluteNextURL(t *testing.T) {
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
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/items?page=2>; rel="next"`, srv.URL))
			_ = json.NewEncoder(w).Encode([]item{{ID: "1"}})
		case 2:
			_ = json.NewEncoder(w).Encode([]item{{ID: "2"}})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var items []item
	meta, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 0, 100, &items)
	if err != nil {
		t.Fatalf("paginateInto() error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if meta.RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestPaginateInto_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "1"}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var items []map[string]string
	_, err := paginateInto(ctx, c, "/api/v1/items", nil, 0, 100, &items)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestPaginateInto_NoLinkHeaderStops(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// No Link header — should stop after first page.
		_ = json.NewEncoder(w).Encode([]item{{ID: "1"}, {ID: "2"}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var items []item
	meta, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 0, 100, &items)
	if err != nil {
		t.Fatalf("paginateInto() error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if meta.RequestCount != 1 {
		t.Errorf("RequestCount = %d, want 1", meta.RequestCount)
	}
}

func TestPaginateInto_LimitReachedAfterPage(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/items?page=2>; rel="next"`, srv.URL))
		_ = json.NewEncoder(w).Encode([]item{{ID: "1"}, {ID: "2"}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var items []item
	// limit == page size, so after first page we hit the limit and stop.
	meta, err := paginateInto(context.Background(), c, "/api/v1/items", nil, 2, 100, &items)
	if err != nil {
		t.Fatalf("paginateInto() error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if meta.RequestCount != 1 {
		t.Errorf("RequestCount = %d, want 1 (should stop at limit)", meta.RequestCount)
	}
}
