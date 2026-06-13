package canvas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUpdateAssignment(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		dueAt := "2026-06-30T23:59:00Z"
		json.NewEncoder(w).Encode(Assignment{
			ID:              "301",
			CourseID:        "42",
			Name:            "Essay 1",
			DueAt:           &dueAt,
			PointsPossible:  100,
			SubmissionTypes: []string{"online_text_entry"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	updates := map[string]any{
		"due_at":          "2026-06-30T23:59:00Z",
		"points_possible": 100,
	}

	a, err := UpdateAssignment(context.Background(), c, "42", "301", updates)
	if err != nil {
		t.Fatalf("UpdateAssignment() error: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/courses/42/assignments/301"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["assignment"]["due_at"] != "2026-06-30T23:59:00Z" {
		t.Errorf("due_at = %v, want %q", gotBody["assignment"]["due_at"], "2026-06-30T23:59:00Z")
	}
	if a.ID != "301" {
		t.Errorf("a.ID = %q, want %q", a.ID, "301")
	}
	if a.DueAt == nil || *a.DueAt != "2026-06-30T23:59:00Z" {
		t.Errorf("a.DueAt = %v, want 2026-06-30T23:59:00Z", a.DueAt)
	}
}

func TestUpdatePage(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		json.NewEncoder(w).Encode(Page{
			URL:       "syllabus",
			Title:     "Syllabus",
			Body:      "<p>Updated syllabus content</p>",
			Published: true,
			CreatedAt: "2026-01-15T10:00:00Z",
			UpdatedAt: "2026-06-12T14:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	updates := map[string]any{
		"body": "<p>Updated syllabus content</p>",
	}

	page, err := UpdatePage(context.Background(), c, "42", "syllabus", updates)
	if err != nil {
		t.Fatalf("UpdatePage() error: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/courses/42/pages/syllabus"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["wiki_page"]["body"] != "<p>Updated syllabus content</p>" {
		t.Errorf("body = %v, want %q", gotBody["wiki_page"]["body"], "<p>Updated syllabus content</p>")
	}
	if page.URL != "syllabus" {
		t.Errorf("page.URL = %q, want %q", page.URL, "syllabus")
	}
	if page.Body != "<p>Updated syllabus content</p>" {
		t.Errorf("page.Body = %q, want %q", page.Body, "<p>Updated syllabus content</p>")
	}
}

func TestPublishModule(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		json.NewEncoder(w).Encode(Module{
			ID:            "10",
			Name:          "Week 1",
			Position:      1,
			Published:     true,
			ItemsCount:    5,
			WorkflowState: "active",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	mod, err := PublishModule(context.Background(), c, "42", "10", true)
	if err != nil {
		t.Fatalf("PublishModule() error: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/courses/42/modules/10"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["module"]["published"] != true {
		t.Errorf("published = %v, want true", gotBody["module"]["published"])
	}
	if !mod.Published {
		t.Error("mod.Published should be true")
	}
	if mod.ID != "10" {
		t.Errorf("mod.ID = %q, want %q", mod.ID, "10")
	}
}

func TestUnpublishModule(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		json.NewEncoder(w).Encode(Module{
			ID:            "10",
			Name:          "Week 1",
			Position:      1,
			Published:     false,
			ItemsCount:    5,
			WorkflowState: "unpublished",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	mod, err := PublishModule(context.Background(), c, "42", "10", false)
	if err != nil {
		t.Fatalf("PublishModule(false) error: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/api/v1/courses/42/modules/10"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["module"]["published"] != false {
		t.Errorf("published = %v, want false", gotBody["module"]["published"])
	}
	if mod.Published {
		t.Error("mod.Published should be false")
	}
}
