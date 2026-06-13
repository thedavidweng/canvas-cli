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

func TestListEnrollments(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		score := 92.5
		json.NewEncoder(w).Encode([]Enrollment{
			{
				ID:       "100",
				UserID:   "789",
				CourseID: "42",
				Type:     "StudentEnrollment",
				Role:     "StudentEnrollment",
				Grades:   &Grades{CurrentScore: &score},
				User:     &User{ID: "789", Name: "Alice Smith", SortableName: "Smith, Alice"},
			},
			{
				ID:       "101",
				UserID:   "790",
				CourseID: "42",
				Type:     "StudentEnrollment",
				Role:     "StudentEnrollment",
				User:     &User{ID: "790", Name: "Bob Jones", SortableName: "Jones, Bob"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	enrollments, meta, err := ListEnrollments(context.Background(), c, "42", RequestOptions{})
	if err != nil {
		t.Fatalf("ListEnrollments() error: %v", err)
	}

	if gotPath != "/api/v1/courses/42/enrollments" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/42/enrollments")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}
	if parsed.Get("include[]") != "total_scores" {
		t.Errorf("include[] = %q, want %q", parsed.Get("include[]"), "total_scores")
	}

	if len(enrollments) != 2 {
		t.Fatalf("len(enrollments) = %d, want 2", len(enrollments))
	}
	if enrollments[0].ID != "100" {
		t.Errorf("enrollments[0].ID = %q, want %q", enrollments[0].ID, "100")
	}
	if enrollments[0].Role != "StudentEnrollment" {
		t.Errorf("enrollments[0].Role = %q, want %q", enrollments[0].Role, "StudentEnrollment")
	}
	if enrollments[0].Grades == nil || enrollments[0].Grades.CurrentScore == nil || *enrollments[0].Grades.CurrentScore != 92.5 {
		t.Errorf("enrollments[0].Grades.CurrentScore = %v, want 92.5", enrollments[0].Grades)
	}
	if enrollments[0].User == nil {
		t.Fatal("enrollments[0].User should not be nil")
	}
	if enrollments[0].User.Name != "Alice Smith" {
		t.Errorf("enrollments[0].User.Name = %q, want %q", enrollments[0].User.Name, "Alice Smith")
	}
	if enrollments[1].User == nil {
		t.Fatal("enrollments[1].User should not be nil")
	}
	if enrollments[1].User.Name != "Bob Jones" {
		t.Errorf("enrollments[1].User.Name = %q, want %q", enrollments[1].User.Name, "Bob Jones")
	}

	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
}

func TestListEnrollmentsPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/42/enrollments?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]Enrollment{
				{ID: "100", UserID: "789", User: &User{ID: "789", Name: "Alice"}},
			})
		case 2:
			json.NewEncoder(w).Encode([]Enrollment{
				{ID: "101", UserID: "790", User: &User{ID: "790", Name: "Bob"}},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	enrollments, meta, err := ListEnrollments(context.Background(), c, "42", RequestOptions{})
	if err != nil {
		t.Fatalf("ListEnrollments() error: %v", err)
	}

	if len(enrollments) != 2 {
		t.Fatalf("len(enrollments) = %d, want 2", len(enrollments))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestListEnrollmentsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, _, err := ListEnrollments(context.Background(), c, "42", RequestOptions{})
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}
