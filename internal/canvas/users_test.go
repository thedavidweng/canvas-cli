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

func TestListUsers(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]User{
			{ID: "789", Name: "Alice Smith", SortableName: "Smith, Alice", LoginID: "alice@example.edu"},
			{ID: "790", Name: "Bob Jones", SortableName: "Jones, Bob", LoginID: "bob@example.edu"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	users, meta, err := ListUsers(context.Background(), c, "42", RequestOptions{})
	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}

	if gotPath != "/api/v1/courses/42/users" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/42/users")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}
	if parsed.Get("enrollment_type[]") != "student" {
		t.Errorf("enrollment_type[] = %q, want %q", parsed.Get("enrollment_type[]"), "student")
	}

	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	if users[0].ID != "789" {
		t.Errorf("users[0].ID = %q, want %q", users[0].ID, "789")
	}
	if users[0].Name != "Alice Smith" {
		t.Errorf("users[0].Name = %q, want %q", users[0].Name, "Alice Smith")
	}
	if users[0].LoginID != "alice@example.edu" {
		t.Errorf("users[0].LoginID = %q, want %q", users[0].LoginID, "alice@example.edu")
	}
	if users[1].Name != "Bob Jones" {
		t.Errorf("users[1].Name = %q, want %q", users[1].Name, "Bob Jones")
	}

	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
}

func TestListUsersPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/42/users?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]User{
				{ID: "789", Name: "Alice"},
			})
		case 2:
			json.NewEncoder(w).Encode([]User{
				{ID: "790", Name: "Bob"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	users, meta, err := ListUsers(context.Background(), c, "42", RequestOptions{})
	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestListUsersError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, _, err := ListUsers(context.Background(), c, "42", RequestOptions{})
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}
