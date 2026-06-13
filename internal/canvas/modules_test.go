package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestListModules(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Module{
			{ID: "1", Name: "Week 1", Position: 1, Published: true, ItemsCount: 5},
			{ID: "2", Name: "Week 2", Position: 2, Published: true, ItemsCount: 3},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	modules, meta, err := ListModules(context.Background(), c, "1", nil)
	if err != nil {
		t.Fatalf("ListModules() error: %v", err)
	}

	if gotPath != "/api/v1/courses/1/modules" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/1/modules")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}

	if len(modules) != 2 {
		t.Fatalf("len(modules) = %d, want 2", len(modules))
	}
	if modules[0].ID != "1" {
		t.Errorf("modules[0].ID = %q, want %q", modules[0].ID, "1")
	}
	if modules[0].Name != "Week 1" {
		t.Errorf("modules[0].Name = %q, want %q", modules[0].Name, "Week 1")
	}
	if modules[0].ItemsCount != 5 {
		t.Errorf("modules[0].ItemsCount = %d, want 5", modules[0].ItemsCount)
	}

	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
	if meta.RequestCount < 1 {
		t.Errorf("meta.RequestCount = %d, want >= 1", meta.RequestCount)
	}
}

func TestListModulesWithQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Module{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	opts := url.Values{"include[]": {"items"}}
	ListModules(context.Background(), c, "1", opts)

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("include[]") != "items" {
		t.Errorf("include[] = %q, want %q", parsed.Get("include[]"), "items")
	}
}

func TestListModulesPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/1/modules?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]Module{
				{ID: "1", Name: "Module 1"},
			})
		case 2:
			json.NewEncoder(w).Encode([]Module{
				{ID: "2", Name: "Module 2"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	modules, meta, err := ListModules(context.Background(), c, "1", nil)
	if err != nil {
		t.Fatalf("ListModules() error: %v", err)
	}

	if len(modules) != 2 {
		t.Fatalf("len(modules) = %d, want 2", len(modules))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestListModuleItems(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ModuleItem{
			{ID: "10", ModuleID: "1", Title: "Read Chapter 1", Type: "Page", Position: 1},
			{ID: "11", ModuleID: "1", Title: "Quiz 1", Type: "Quiz", Position: 2},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := ListModuleItems(context.Background(), c, "1", "3", nil)
	if err != nil {
		t.Fatalf("ListModuleItems() error: %v", err)
	}

	if gotPath != "/api/v1/courses/1/modules/3/items" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/1/modules/3/items")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "10" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "10")
	}
	if items[0].Title != "Read Chapter 1" {
		t.Errorf("items[0].Title = %q, want %q", items[0].Title, "Read Chapter 1")
	}
	if items[0].Type != "Page" {
		t.Errorf("items[0].Type = %q, want %q", items[0].Type, "Page")
	}
	if items[1].Type != "Quiz" {
		t.Errorf("items[1].Type = %q, want %q", items[1].Type, "Quiz")
	}

	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
}

func TestListModuleItemsPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/1/modules/1/items?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]ModuleItem{
				{ID: "10", Title: "Item 1"},
			})
		case 2:
			json.NewEncoder(w).Encode([]ModuleItem{
				{ID: "11", Title: "Item 2"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	items, meta, err := ListModuleItems(context.Background(), c, "1", "1", nil)
	if err != nil {
		t.Fatalf("ListModuleItems() error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestListModuleItemsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"errors":[{"message":"not found"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, _, err := ListModuleItems(context.Background(), c, "1", "999", nil)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}
