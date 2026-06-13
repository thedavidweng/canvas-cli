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

func TestListDecodesSliceOfItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Course{
			{ID: "1", Name: "Algebra 101", CourseCode: "MATH101"},
			{ID: "2", Name: "History 201", CourseCode: "HIST201"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	courses, meta, err := List[Course](context.Background(), c, "/api/v1/courses", nil, 100)
	if err != nil {
		t.Fatalf("List[Course]() error: %v", err)
	}
	if len(courses) != 2 {
		t.Fatalf("len(courses) = %d, want 2", len(courses))
	}
	if courses[0].ID != "1" {
		t.Errorf("courses[0].ID = %q, want %q", courses[0].ID, "1")
	}
	if courses[0].Name != "Algebra 101" {
		t.Errorf("courses[0].Name = %q, want %q", courses[0].Name, "Algebra 101")
	}
	if courses[1].ID != "2" {
		t.Errorf("courses[1].ID = %q, want %q", courses[1].ID, "2")
	}
	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestListHandlesPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]Course{{ID: "1", Name: "Page1 Course"}})
		case 2:
			json.NewEncoder(w).Encode([]Course{{ID: "2", Name: "Page2 Course"}})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	courses, meta, err := List[Course](context.Background(), c, "/api/v1/courses", nil, 100)
	if err != nil {
		t.Fatalf("List[Course]() error: %v", err)
	}
	if len(courses) != 2 {
		t.Fatalf("len(courses) = %d, want 2", len(courses))
	}
	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestListPassesQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Course{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	query := url.Values{"enrollment_state": {"active"}}

	_, _, err := List[Course](context.Background(), c, "/api/v1/courses", query, 50)
	if err != nil {
		t.Fatalf("List[Course]() error: %v", err)
	}

	if gotQuery == "" {
		t.Fatal("expected query params, got empty")
	}
	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("enrollment_state") != "active" {
		t.Errorf("enrollment_state = %q, want %q", parsed.Get("enrollment_state"), "active")
	}
	if parsed.Get("per_page") != "50" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "50")
	}
}

func TestListReturnsEmptySliceForEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Course{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	courses, _, err := List[Course](context.Background(), c, "/api/v1/courses", nil, 100)
	if err != nil {
		t.Fatalf("List[Course]() error: %v", err)
	}
	if len(courses) != 0 {
		t.Errorf("len(courses) = %d, want 0", len(courses))
	}
}

func TestListHandlesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid access token"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, _, err := List[Course](context.Background(), c, "/api/v1/courses", nil, 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetDecodesSingleItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Course{ID: "42", Name: "Advanced Go", CourseCode: "CS301"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	course, err := Get[Course](context.Background(), c, "/api/v1/courses/42")
	if err != nil {
		t.Fatalf("Get[Course]() error: %v", err)
	}
	if course.ID != "42" {
		t.Errorf("course.ID = %q, want %q", course.ID, "42")
	}
	if course.Name != "Advanced Go" {
		t.Errorf("course.Name = %q, want %q", course.Name, "Advanced Go")
	}
	if course.CourseCode != "CS301" {
		t.Errorf("course.CourseCode = %q, want %q", course.CourseCode, "CS301")
	}
}

func TestGetHandlesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Course not found"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := Get[Course](context.Background(), c, "/api/v1/courses/999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSendsCorrectPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Course{ID: "5", Name: "Physics"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := Get[Course](context.Background(), c, "/api/v1/courses/5")
	if err != nil {
		t.Fatalf("Get[Course]() error: %v", err)
	}
	if gotPath != "/api/v1/courses/5" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/5")
	}
}

func TestListWorksWithDifferentTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Assignment{
			{ID: "10", Name: "Homework 1", CourseID: "5"},
			{ID: "20", Name: "Midterm", CourseID: "5"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	assignments, _, err := List[Assignment](context.Background(), c, "/api/v1/courses/5/assignments", nil, 100)
	if err != nil {
		t.Fatalf("List[Assignment]() error: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("len(assignments) = %d, want 2", len(assignments))
	}
	if assignments[0].ID != "10" {
		t.Errorf("assignments[0].ID = %q, want %q", assignments[0].ID, "10")
	}
	if assignments[0].Name != "Homework 1" {
		t.Errorf("assignments[0].Name = %q, want %q", assignments[0].Name, "Homework 1")
	}
}

func TestGetWorksWithDifferentTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Assignment{ID: "10", Name: "Homework 1", CourseID: "5", PointsPossible: 100})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	assignment, err := Get[Assignment](context.Background(), c, "/api/v1/courses/5/assignments/10")
	if err != nil {
		t.Fatalf("Get[Assignment]() error: %v", err)
	}
	if assignment.ID != "10" {
		t.Errorf("assignment.ID = %q, want %q", assignment.ID, "10")
	}
	if assignment.PointsPossible != 100 {
		t.Errorf("assignment.PointsPossible = %f, want 100", assignment.PointsPossible)
	}
}
