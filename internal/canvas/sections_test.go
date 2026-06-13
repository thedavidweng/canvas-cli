package canvas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestListSections(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		total := 25
		json.NewEncoder(w).Encode([]Section{
			{ID: "10", Name: "Section A", CourseID: "42", TotalStudents: &total},
			{ID: "11", Name: "Section B", CourseID: "42"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	sections, err := ListSections(context.Background(), c, "42")
	if err != nil {
		t.Fatalf("ListSections() error: %v", err)
	}

	if gotPath != "/api/v1/courses/42/sections" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/42/sections")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}
	if parsed.Get("include[]") != "total_students" {
		t.Errorf("include[] = %q, want %q", parsed.Get("include[]"), "total_students")
	}

	if len(sections) != 2 {
		t.Fatalf("len(sections) = %d, want 2", len(sections))
	}
	if sections[0].ID != "10" {
		t.Errorf("sections[0].ID = %q, want %q", sections[0].ID, "10")
	}
	if sections[0].Name != "Section A" {
		t.Errorf("sections[0].Name = %q, want %q", sections[0].Name, "Section A")
	}
	if sections[0].TotalStudents == nil || *sections[0].TotalStudents != 25 {
		t.Errorf("sections[0].TotalStudents = %v, want 25", sections[0].TotalStudents)
	}
	if sections[1].ID != "11" {
		t.Errorf("sections[1].ID = %q, want %q", sections[1].ID, "11")
	}
	if sections[1].Name != "Section B" {
		t.Errorf("sections[1].Name = %q, want %q", sections[1].Name, "Section B")
	}
}

func TestListSectionsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, err := ListSections(context.Background(), c, "42")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}
