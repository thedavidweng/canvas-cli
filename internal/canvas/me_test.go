package canvas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetActivityStream(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ActivityItem{
			{
				ID:        "1",
				Title:     "New submission",
				Message:   "Alice submitted Homework 1",
				Type:      "Submission",
				ReadState: "unread",
				CreatedAt: "2026-06-15T10:00:00Z",
			},
			{
				ID:        "2",
				Title:     "Grade posted",
				Type:      "Grade",
				ReadState: "read",
				CreatedAt: "2026-06-14T09:00:00Z",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	items, err := GetActivityStream(context.Background(), c)
	if err != nil {
		t.Fatalf("GetActivityStream() error: %v", err)
	}

	wantPath := "/api/v1/users/self/activity_stream"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "1" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "1")
	}
	if items[0].Title != "New submission" {
		t.Errorf("items[0].Title = %q, want %q", items[0].Title, "New submission")
	}
	if items[0].Type != "Submission" {
		t.Errorf("items[0].Type = %q, want %q", items[0].Type, "Submission")
	}
	if items[1].ID != "2" {
		t.Errorf("items[1].ID = %q, want %q", items[1].ID, "2")
	}
}

func TestGetActivityStreamServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GetActivityStream(context.Background(), c)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestGetTodoItems(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		due := "2026-06-20T23:59:00Z"
		json.NewEncoder(w).Encode([]TodoItem{
			{
				ID:            "10",
				ContextCode:   "course_42",
				Title:         "Read Chapter 5",
				Type:          "submitting",
				DueDate:       &due,
				WorkflowState: "published",
			},
			{
				ID:            "11",
				ContextCode:   "course_42",
				Title:         "Quiz 3",
				Type:          "quiz",
				WorkflowState: "published",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	items, err := GetTodoItems(context.Background(), c)
	if err != nil {
		t.Fatalf("GetTodoItems() error: %v", err)
	}

	wantPath := "/api/v1/users/self/todo"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "10" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "10")
	}
	if items[0].Title != "Read Chapter 5" {
		t.Errorf("items[0].Title = %q, want %q", items[0].Title, "Read Chapter 5")
	}
	if items[0].DueDate == nil || *items[0].DueDate != "2026-06-20T23:59:00Z" {
		t.Errorf("items[0].DueDate = %v, want %q", items[0].DueDate, "2026-06-20T23:59:00Z")
	}
	if items[1].Type != "quiz" {
		t.Errorf("items[1].Type = %q, want %q", items[1].Type, "quiz")
	}
}

func TestGetTodoItemsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GetTodoItems(context.Background(), c)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestGetUpcomingEvents(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]UpcomingEvent{
			{
				ID:          "100",
				Title:       "Midterm Exam",
				StartAt:     "2026-06-20T09:00:00Z",
				EndAt:       "2026-06-20T11:00:00Z",
				ContextCode: "course_42",
				Type:        "calendar_event",
			},
			{
				ID:          "101",
				Title:       "Project Presentation",
				StartAt:     "2026-06-25T14:00:00Z",
				ContextCode: "course_42",
				Type:        "calendar_event",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	events, err := GetUpcomingEvents(context.Background(), c)
	if err != nil {
		t.Fatalf("GetUpcomingEvents() error: %v", err)
	}

	wantPath := "/api/v1/users/self/upcoming_events"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].ID != "100" {
		t.Errorf("events[0].ID = %q, want %q", events[0].ID, "100")
	}
	if events[0].Title != "Midterm Exam" {
		t.Errorf("events[0].Title = %q, want %q", events[0].Title, "Midterm Exam")
	}
	if events[0].StartAt != "2026-06-20T09:00:00Z" {
		t.Errorf("events[0].StartAt = %q, want %q", events[0].StartAt, "2026-06-20T09:00:00Z")
	}
	if events[1].ID != "101" {
		t.Errorf("events[1].ID = %q, want %q", events[1].ID, "101")
	}
}

func TestGetUpcomingEventsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GetUpcomingEvents(context.Background(), c)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}
