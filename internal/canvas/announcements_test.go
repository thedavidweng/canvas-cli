package canvas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestListAnnouncements(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]DiscussionTopic{
			{
				ID:             "10",
				Title:          "Exam moved to Friday",
				Message:        "The midterm has been rescheduled.",
				PostedAt:       strPtr("2026-06-01T10:00:00Z"),
				IsAnnouncement: true,
			},
			{
				ID:             "11",
				Title:          "Office hours cancelled",
				Message:        "No office hours this week.",
				PostedAt:       strPtr("2026-06-10T09:00:00Z"),
				IsAnnouncement: true,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	opts := url.Values{}
	announcements, meta, err := ListAnnouncements(context.Background(), c, "42", opts)
	if err != nil {
		t.Fatalf("ListAnnouncements() error: %v", err)
	}

	wantPath := "/api/v1/announcements"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	// Verify context_codes[]=course_42 was sent
	ctxCodes := gotQuery["context_codes[]"]
	found := false
	for _, code := range ctxCodes {
		if code == "course_42" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("context_codes[] does not contain course_42, got: %v", ctxCodes)
	}

	if len(announcements) != 2 {
		t.Fatalf("len(announcements) = %d, want 2", len(announcements))
	}
	if announcements[0].Title != "Exam moved to Friday" {
		t.Errorf("title = %q, want %q", announcements[0].Title, "Exam moved to Friday")
	}
	if !announcements[0].IsAnnouncement {
		t.Error("first announcement IsAnnouncement should be true")
	}

	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestListAnnouncementsIncludesContextCodes(t *testing.T) {
	var rawQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]DiscussionTopic{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	opts := url.Values{}

	_, _, err := ListAnnouncements(context.Background(), c, "99", opts)
	if err != nil {
		t.Fatalf("ListAnnouncements() error: %v", err)
	}

	if !strings.Contains(rawQuery, "context_codes%5B%5D=course_99") && !strings.Contains(rawQuery, "context_codes[]=course_99") {
		t.Errorf("query does not contain context_codes[]=course_99, got: %s", rawQuery)
	}
}

func TestCreateAnnouncement(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&gotBody)

		postedAt := "2026-06-12T10:00:00Z"
		json.NewEncoder(w).Encode(DiscussionTopic{
			ID:             "200",
			Title:          "Final exam details",
			Message:        "The final will be held in Room 101.",
			PostedAt:       &postedAt,
			IsAnnouncement: true,
			Published:      true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	topic, err := CreateAnnouncement(context.Background(), c, "42", "Final exam details", "The final will be held in Room 101.")
	if err != nil {
		t.Fatalf("CreateAnnouncement() error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/api/v1/courses/42/discussion_topics"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["is_announcement"] != true {
		t.Errorf("is_announcement = %v, want true", gotBody["is_announcement"])
	}
	if gotBody["title"] != "Final exam details" {
		t.Errorf("title = %v, want %q", gotBody["title"], "Final exam details")
	}
	if gotBody["message"] != "The final will be held in Room 101." {
		t.Errorf("message = %v, want %q", gotBody["message"], "The final will be held in Room 101.")
	}
	if topic.ID != "200" {
		t.Errorf("topic.ID = %q, want %q", topic.ID, "200")
	}
	if !topic.IsAnnouncement {
		t.Error("topic.IsAnnouncement should be true")
	}
}

func strPtr(s string) *string {
	return &s
}
