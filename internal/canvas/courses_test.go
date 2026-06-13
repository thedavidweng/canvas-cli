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

func TestListCourses(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Course{
			{ID: "1", Name: "Course A", CourseCode: "CA101", WorkflowState: "available"},
			{ID: "2", Name: "Course B", CourseCode: "CB201", WorkflowState: "available"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	opts := url.Values{"enrollment_state": {"active"}}
	courses, meta, err := ListCourses(context.Background(), c, opts)
	if err != nil {
		t.Fatalf("ListCourses() error: %v", err)
	}

	if gotPath != "/api/v1/courses" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("enrollment_state") != "active" {
		t.Errorf("enrollment_state = %q, want %q", parsed.Get("enrollment_state"), "active")
	}
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}

	if len(courses) != 2 {
		t.Fatalf("len(courses) = %d, want 2", len(courses))
	}
	if courses[0].ID != "1" {
		t.Errorf("courses[0].ID = %q, want %q", courses[0].ID, "1")
	}
	if courses[0].Name != "Course A" {
		t.Errorf("courses[0].Name = %q, want %q", courses[0].Name, "Course A")
	}
	if courses[1].ID != "2" {
		t.Errorf("courses[1].ID = %q, want %q", courses[1].ID, "2")
	}

	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
	if meta.RequestCount < 1 {
		t.Errorf("meta.RequestCount = %d, want >= 1", meta.RequestCount)
	}
}

func TestListCoursesPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]Course{
				{ID: "1", Name: "Course A"},
			})
		case 2:
			json.NewEncoder(w).Encode([]Course{
				{ID: "2", Name: "Course B"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	courses, meta, err := ListCourses(context.Background(), c, nil)
	if err != nil {
		t.Fatalf("ListCourses() error: %v", err)
	}

	if len(courses) != 2 {
		t.Fatalf("len(courses) = %d, want 2", len(courses))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestGetCourse(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Course{
			ID:               "42",
			Name:             "Advanced Go",
			CourseCode:       "CS401",
			WorkflowState:    "available",
			EnrollmentTermID: "1",
			Term:             &Term{ID: "1", Name: "Fall 2025"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	course, err := GetCourse(context.Background(), c, "42", nil)
	if err != nil {
		t.Fatalf("GetCourse() error: %v", err)
	}

	if gotPath != "/api/v1/courses/42" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/42")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("include[]") != "term" {
		t.Errorf("include[] = %q, want %q", parsed.Get("include[]"), "term")
	}

	if course.ID != "42" {
		t.Errorf("ID = %q, want %q", course.ID, "42")
	}
	if course.Name != "Advanced Go" {
		t.Errorf("Name = %q, want %q", course.Name, "Advanced Go")
	}
	if course.Term == nil {
		t.Fatal("Term should not be nil")
	}
	if course.Term.Name != "Fall 2025" {
		t.Errorf("Term.Name = %q, want %q", course.Term.Name, "Fall 2025")
	}
}

func TestGetCourseWithExtraQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Course{ID: "42", Name: "Test"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	opts := url.Values{"include[]": {"total_scores"}}
	_, err := GetCourse(context.Background(), c, "42", opts)
	if err != nil {
		t.Fatalf("GetCourse() error: %v", err)
	}

	parsed, _ := url.ParseQuery(gotQuery)
	includes := parsed["include[]"]
	hasTerm := false
	hasTotalScores := false
	for _, inc := range includes {
		if inc == "term" {
			hasTerm = true
		}
		if inc == "total_scores" {
			hasTotalScores = true
		}
	}
	if !hasTerm {
		t.Error("include[] should contain 'term'")
	}
	if !hasTotalScores {
		t.Error("include[] should contain 'total_scores'")
	}
}

func TestGetCourseNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"errors":[{"message":"not found"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, err := GetCourse(context.Background(), c, "999", nil)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestListCourseTabs(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Tab{
			{ID: "home", Label: "Home", Type: "internal", HTMLURL: "/courses/1", FullURL: "https://canvas.example.com/courses/1", Position: 1, Visibility: "public"},
			{ID: "modules", Label: "Modules", Type: "internal", HTMLURL: "/courses/1/modules", FullURL: "https://canvas.example.com/courses/1/modules", Position: 2, Visibility: "public"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	tabs, err := ListCourseTabs(context.Background(), c, "1")
	if err != nil {
		t.Fatalf("ListCourseTabs() error: %v", err)
	}

	if gotPath != "/api/v1/courses/1/tabs" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/1/tabs")
	}

	if len(tabs) != 2 {
		t.Fatalf("len(tabs) = %d, want 2", len(tabs))
	}
	if tabs[0].ID != "home" {
		t.Errorf("tabs[0].ID = %q, want %q", tabs[0].ID, "home")
	}
	if tabs[0].Label != "Home" {
		t.Errorf("tabs[0].Label = %q, want %q", tabs[0].Label, "Home")
	}
	if tabs[0].Type != "internal" {
		t.Errorf("tabs[0].Type = %q, want %q", tabs[0].Type, "internal")
	}
	if tabs[0].Position != 1 {
		t.Errorf("tabs[0].Position = %d, want 1", tabs[0].Position)
	}
	if tabs[0].Visibility != "public" {
		t.Errorf("tabs[0].Visibility = %q, want %q", tabs[0].Visibility, "public")
	}
	if tabs[1].ID != "modules" {
		t.Errorf("tabs[1].ID = %q, want %q", tabs[1].ID, "modules")
	}
}

func TestListCourseTabsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"errors":[{"message":"not found"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, err := ListCourseTabs(context.Background(), c, "999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}
