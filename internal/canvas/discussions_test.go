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

func TestListDiscussions(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]DiscussionTopic{
			{
				ID:             "201",
				Title:          "Week 1 discussion",
				Message:        "Introduce yourselves!",
				DiscussionType: "threaded",
			},
			{
				ID:             "202",
				Title:          "Week 2 discussion",
				Message:        "Discuss the reading.",
				DiscussionType: "threaded",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	discussions, meta, err := ListDiscussions(context.Background(), c, "42", nil)
	if err != nil {
		t.Fatalf("ListDiscussions() error: %v", err)
	}

	wantPath := "/api/v1/courses/42/discussion_topics"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if len(discussions) != 2 {
		t.Fatalf("len(discussions) = %d, want 2", len(discussions))
	}
	if discussions[0].ID != "201" {
		t.Errorf("discussions[0].ID = %q, want %q", discussions[0].ID, "201")
	}
	if discussions[0].Title != "Week 1 discussion" {
		t.Errorf("discussions[0].Title = %q, want %q", discussions[0].Title, "Week 1 discussion")
	}
	if discussions[1].DiscussionType != "threaded" {
		t.Errorf("discussions[1].DiscussionType = %q, want %q", discussions[1].DiscussionType, "threaded")
	}

	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestListDiscussionsPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/42/discussion_topics?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]DiscussionTopic{
				{ID: "1", Title: "Topic 1"},
				{ID: "2", Title: "Topic 2"},
			})
		case 2:
			json.NewEncoder(w).Encode([]DiscussionTopic{
				{ID: "3", Title: "Topic 3"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	discussions, meta, err := ListDiscussions(context.Background(), c, "42", nil)
	if err != nil {
		t.Fatalf("ListDiscussions() error: %v", err)
	}

	if len(discussions) != 3 {
		t.Errorf("len(discussions) = %d, want 3", len(discussions))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestGetDiscussion(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DiscussionTopic{
			ID:             "201",
			Title:          "Week 1 discussion",
			Message:        "Introduce yourselves!",
			DiscussionType: "threaded",
			Published:      true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	topic, err := GetDiscussion(context.Background(), c, "42", "201")
	if err != nil {
		t.Fatalf("GetDiscussion() error: %v", err)
	}

	wantPath := "/api/v1/courses/42/discussion_topics/201"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if topic.ID != "201" {
		t.Errorf("topic.ID = %q, want %q", topic.ID, "201")
	}
	if topic.Title != "Week 1 discussion" {
		t.Errorf("topic.Title = %q, want %q", topic.Title, "Week 1 discussion")
	}
	if topic.Message != "Introduce yourselves!" {
		t.Errorf("topic.Message = %q, want %q", topic.Message, "Introduce yourselves!")
	}
	if !topic.Published {
		t.Error("topic.Published should be true")
	}
}

func TestListDiscussionEntries(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		parentID := "901"
		json.NewEncoder(w).Encode([]DiscussionEntry{
			{
				ID:        "901",
				UserID:    "789",
				Message:   "Great topic!",
				CreatedAt: "2026-06-11T16:00:00Z",
			},
			{
				ID:        "902",
				UserID:    "790",
				Message:   "I agree!",
				CreatedAt: "2026-06-11T16:30:00Z",
				ParentID:  &parentID,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	entries, meta, err := ListDiscussionEntries(context.Background(), c, "42", "201", nil)
	if err != nil {
		t.Fatalf("ListDiscussionEntries() error: %v", err)
	}

	wantPath := "/api/v1/courses/42/discussion_topics/201/entries"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].ID != "901" {
		t.Errorf("entries[0].ID = %q, want %q", entries[0].ID, "901")
	}
	if entries[0].Message != "Great topic!" {
		t.Errorf("entries[0].Message = %q, want %q", entries[0].Message, "Great topic!")
	}
	if entries[1].ParentID == nil || *entries[1].ParentID != "901" {
		t.Errorf("entries[1].ParentID = %v, want %q", entries[1].ParentID, "901")
	}

	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestListDiscussionEntriesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, _, err := ListDiscussionEntries(context.Background(), c, "42", "201", nil)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestCreateDiscussion(t *testing.T) {
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

		postedAt := "2026-06-15T10:00:00Z"
		json.NewEncoder(w).Encode(DiscussionTopic{
			ID:             "301",
			Title:          "Week 3 discussion",
			Message:        "Discuss the reading.",
			PostedAt:       &postedAt,
			DiscussionType: "threaded",
			Published:      true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	topic, err := CreateDiscussion(context.Background(), c, "42", "Week 3 discussion", "Discuss the reading.")
	if err != nil {
		t.Fatalf("CreateDiscussion() error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/api/v1/courses/42/discussion_topics"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["title"] != "Week 3 discussion" {
		t.Errorf("title = %v, want %q", gotBody["title"], "Week 3 discussion")
	}
	if gotBody["message"] != "Discuss the reading." {
		t.Errorf("message = %v, want %q", gotBody["message"], "Discuss the reading.")
	}

	if topic.ID != "301" {
		t.Errorf("topic.ID = %q, want %q", topic.ID, "301")
	}
	if topic.Title != "Week 3 discussion" {
		t.Errorf("topic.Title = %q, want %q", topic.Title, "Week 3 discussion")
	}
	if topic.DiscussionType != "threaded" {
		t.Errorf("topic.DiscussionType = %q, want %q", topic.DiscussionType, "threaded")
	}
	if !topic.Published {
		t.Error("topic.Published should be true")
	}
}

func TestCreateDiscussionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Forbidden"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := CreateDiscussion(context.Background(), c, "42", "Test", "Test message")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

func TestReplyToDiscussion(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(DiscussionEntry{
			ID:        "901",
			UserID:    "789",
			Message:   "Great topic!",
			CreatedAt: "2026-06-11T16:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	entry, err := ReplyToDiscussion(context.Background(), c, "42", "201", "Great topic!")
	if err != nil {
		t.Fatalf("ReplyToDiscussion() error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/api/v1/courses/42/discussion_topics/201/entries"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["message"] != "Great topic!" {
		t.Errorf("message = %q, want %q", gotBody["message"], "Great topic!")
	}
	if entry.ID != "901" {
		t.Errorf("entry.ID = %q, want %q", entry.ID, "901")
	}
	if entry.UserID != "789" {
		t.Errorf("entry.UserID = %q, want %q", entry.UserID, "789")
	}
	if entry.Message != "Great topic!" {
		t.Errorf("entry.Message = %q, want %q", entry.Message, "Great topic!")
	}
	if entry.CreatedAt != "2026-06-11T16:00:00Z" {
		t.Errorf("entry.CreatedAt = %q, want %q", entry.CreatedAt, "2026-06-11T16:00:00Z")
	}
}

func TestReplyToEntry(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewDecoder(r.Body).Decode(&gotBody)
		parentID := "901"
		json.NewEncoder(w).Encode(DiscussionEntry{
			ID:        "902",
			UserID:    "790",
			Message:   "I agree with you!",
			CreatedAt: "2026-06-11T16:30:00Z",
			ParentID:  &parentID,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	entry, err := ReplyToEntry(context.Background(), c, "42", "201", "901", "I agree with you!")
	if err != nil {
		t.Fatalf("ReplyToEntry() error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/api/v1/courses/42/discussion_topics/201/entries/901/replies"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["message"] != "I agree with you!" {
		t.Errorf("message = %q, want %q", gotBody["message"], "I agree with you!")
	}
	if entry.ID != "902" {
		t.Errorf("entry.ID = %q, want %q", entry.ID, "902")
	}
	if entry.ParentID == nil || *entry.ParentID != "901" {
		t.Errorf("entry.ParentID = %v, want %q", entry.ParentID, "901")
	}
}
