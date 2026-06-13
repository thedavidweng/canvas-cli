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

func TestListAssignments(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Assignment{
			{ID: "10", CourseID: "1", Name: "Homework 1", PointsPossible: 100, SubmissionTypes: []string{"online_text_entry"}},
			{ID: "11", CourseID: "1", Name: "Homework 2", PointsPossible: 50, SubmissionTypes: []string{"online_upload"}},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	assignments, meta, err := ListAssignments(context.Background(), c, "1", nil)
	if err != nil {
		t.Fatalf("ListAssignments() error: %v", err)
	}

	if gotPath != "/api/v1/courses/1/assignments" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/1/assignments")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
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
	if assignments[0].PointsPossible != 100 {
		t.Errorf("assignments[0].PointsPossible = %f, want 100", assignments[0].PointsPossible)
	}
	if assignments[1].ID != "11" {
		t.Errorf("assignments[1].ID = %q, want %q", assignments[1].ID, "11")
	}

	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
}

func TestListAssignmentsWithQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Assignment{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	opts := url.Values{"order_by": {"due_at"}}
	ListAssignments(context.Background(), c, "1", opts)

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("order_by") != "due_at" {
		t.Errorf("order_by = %q, want %q", parsed.Get("order_by"), "due_at")
	}
}

func TestListAssignmentsPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/1/assignments?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]Assignment{
				{ID: "10", Name: "HW 1"},
			})
		case 2:
			json.NewEncoder(w).Encode([]Assignment{
				{ID: "11", Name: "HW 2"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	assignments, meta, err := ListAssignments(context.Background(), c, "1", nil)
	if err != nil {
		t.Fatalf("ListAssignments() error: %v", err)
	}

	if len(assignments) != 2 {
		t.Fatalf("len(assignments) = %d, want 2", len(assignments))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestGetAssignment(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		dueAt := "2025-12-15T23:59:00Z"
		json.NewEncoder(w).Encode(Assignment{
			ID:              "42",
			CourseID:        "1",
			Name:            "Final Project",
			DueAt:           &dueAt,
			PointsPossible:  200,
			SubmissionTypes: []string{"online_upload", "online_text_entry"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	assignment, err := GetAssignment(context.Background(), c, "1", "42")
	if err != nil {
		t.Fatalf("GetAssignment() error: %v", err)
	}

	if gotPath != "/api/v1/courses/1/assignments/42" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/1/assignments/42")
	}

	if assignment.ID != "42" {
		t.Errorf("ID = %q, want %q", assignment.ID, "42")
	}
	if assignment.Name != "Final Project" {
		t.Errorf("Name = %q, want %q", assignment.Name, "Final Project")
	}
	if assignment.PointsPossible != 200 {
		t.Errorf("PointsPossible = %f, want 200", assignment.PointsPossible)
	}
	if assignment.DueAt == nil {
		t.Fatal("DueAt should not be nil")
	}
	if *assignment.DueAt != "2025-12-15T23:59:00Z" {
		t.Errorf("DueAt = %q, want %q", *assignment.DueAt, "2025-12-15T23:59:00Z")
	}
	if len(assignment.SubmissionTypes) != 2 {
		t.Errorf("len(SubmissionTypes) = %d, want 2", len(assignment.SubmissionTypes))
	}
}

func TestGetAssignmentNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"errors":[{"message":"not found"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, err := GetAssignment(context.Background(), c, "1", "999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestListAssignmentGroups(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]AssignmentGroup{
			{
				ID:          "1",
				Name:        "Homework",
				Position:    1,
				GroupWeight: 40,
				Assignments: []Assignment{
					{ID: "10", Name: "HW 1"},
				},
			},
			{
				ID:          "2",
				Name:        "Exams",
				Position:    2,
				GroupWeight: 60,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	groups, err := ListAssignmentGroups(context.Background(), c, "1")
	if err != nil {
		t.Fatalf("ListAssignmentGroups() error: %v", err)
	}

	if gotPath != "/api/v1/courses/1/assignment_groups" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/1/assignment_groups")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}
	if parsed.Get("include[]") != "assignments" {
		t.Errorf("include[] = %q, want %q", parsed.Get("include[]"), "assignments")
	}

	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if groups[0].ID != "1" {
		t.Errorf("groups[0].ID = %q, want %q", groups[0].ID, "1")
	}
	if groups[0].Name != "Homework" {
		t.Errorf("groups[0].Name = %q, want %q", groups[0].Name, "Homework")
	}
	if groups[0].GroupWeight != 40 {
		t.Errorf("groups[0].GroupWeight = %f, want 40", groups[0].GroupWeight)
	}
	if len(groups[0].Assignments) != 1 {
		t.Errorf("len(groups[0].Assignments) = %d, want 1", len(groups[0].Assignments))
	}
	if groups[0].Assignments[0].ID != "10" {
		t.Errorf("groups[0].Assignments[0].ID = %q, want %q", groups[0].Assignments[0].ID, "10")
	}
	if groups[1].Name != "Exams" {
		t.Errorf("groups[1].Name = %q, want %q", groups[1].Name, "Exams")
	}
}

func TestListAssignmentGroupsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"errors":[{"message":"not found"}]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, err := ListAssignmentGroups(context.Background(), c, "999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}
