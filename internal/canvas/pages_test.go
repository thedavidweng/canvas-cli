package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListPages(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Page{
			{
				URL:       "syllabus",
				Title:     "Syllabus",
				Published: true,
				CreatedAt: "2026-01-15T08:00:00Z",
				UpdatedAt: "2026-05-01T12:00:00Z",
			},
			{
				URL:       "resources",
				Title:     "Resources",
				Published: true,
				CreatedAt: "2026-01-20T08:00:00Z",
				UpdatedAt: "2026-06-01T12:00:00Z",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	pages, meta, err := ListPages(context.Background(), c, "42", nil)
	if err != nil {
		t.Fatalf("ListPages() error: %v", err)
	}

	wantPath := "/api/v1/courses/42/pages"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if len(pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(pages))
	}
	if pages[0].URL != "syllabus" {
		t.Errorf("pages[0].URL = %q, want %q", pages[0].URL, "syllabus")
	}
	if pages[0].Title != "Syllabus" {
		t.Errorf("pages[0].Title = %q, want %q", pages[0].Title, "Syllabus")
	}
	if pages[1].URL != "resources" {
		t.Errorf("pages[1].URL = %q, want %q", pages[1].URL, "resources")
	}

	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestListPagesPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/42/pages?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]Page{
				{URL: "page-1", Title: "Page 1"},
				{URL: "page-2", Title: "Page 2"},
			})
		case 2:
			json.NewEncoder(w).Encode([]Page{
				{URL: "page-3", Title: "Page 3"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	pages, meta, err := ListPages(context.Background(), c, "42", nil)
	if err != nil {
		t.Fatalf("ListPages() error: %v", err)
	}

	if len(pages) != 3 {
		t.Errorf("len(pages) = %d, want 3", len(pages))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestGetPage(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Page{
			URL:       "syllabus",
			Title:     "Syllabus",
			Body:      "<h1>Course Syllabus</h1><p>Welcome to the course.</p>",
			Published: true,
			CreatedAt: "2026-01-15T08:00:00Z",
			UpdatedAt: "2026-05-01T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	page, err := GetPage(context.Background(), c, "42", "syllabus")
	if err != nil {
		t.Fatalf("GetPage() error: %v", err)
	}

	wantPath := "/api/v1/courses/42/pages/syllabus"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if page.URL != "syllabus" {
		t.Errorf("page.URL = %q, want %q", page.URL, "syllabus")
	}
	if page.Title != "Syllabus" {
		t.Errorf("page.Title = %q, want %q", page.Title, "Syllabus")
	}
	if page.Body != "<h1>Course Syllabus</h1><p>Welcome to the course.</p>" {
		t.Errorf("page.Body = %q, want %q", page.Body, "<h1>Course Syllabus</h1><p>Welcome to the course.</p>")
	}
	if !page.Published {
		t.Error("page.Published should be true")
	}
}

func TestGetPageURLPathEncoding(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Page{
			URL:   "week-1/notes",
			Title: "Week 1 Notes",
			Body:  "Notes content",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	page, err := GetPage(context.Background(), c, "42", "week-1/notes")
	if err != nil {
		t.Fatalf("GetPage() error: %v", err)
	}

	// The page URL with a slash should be properly encoded in the path
	wantPath := "/api/v1/courses/42/pages/week-1/notes"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if page.Title != "Week 1 Notes" {
		t.Errorf("page.Title = %q, want %q", page.Title, "Week 1 Notes")
	}
}

func TestListPagesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, _, err := ListPages(context.Background(), c, "42", nil)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}
