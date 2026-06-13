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

func TestGetSubmission(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		score := 85.5
		grade := "B+"
		submittedAt := "2026-06-10T23:55:00Z"
		attempt := 2
		json.NewEncoder(w).Encode(Submission{
			ID:            "501",
			UserID:        "789",
			AssignmentID:  "301",
			Score:         &score,
			Grade:         &grade,
			SubmittedAt:   &submittedAt,
			WorkflowState: "submitted",
			Late:          false,
			Missing:       false,
			Attempt:       &attempt,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	sub, err := GetSubmission(context.Background(), c, "42", "301", "789")
	if err != nil {
		t.Fatalf("GetSubmission() error: %v", err)
	}

	wantPath := "/api/v1/courses/42/assignments/301/submissions/789"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if sub.ID != "501" {
		t.Errorf("sub.ID = %q, want %q", sub.ID, "501")
	}
	if sub.UserID != "789" {
		t.Errorf("sub.UserID = %q, want %q", sub.UserID, "789")
	}
	if sub.AssignmentID != "301" {
		t.Errorf("sub.AssignmentID = %q, want %q", sub.AssignmentID, "301")
	}
	if sub.Score == nil || *sub.Score != 85.5 {
		t.Errorf("sub.Score = %v, want 85.5", sub.Score)
	}
	if sub.Grade == nil || *sub.Grade != "B+" {
		t.Errorf("sub.Grade = %v, want B+", sub.Grade)
	}
	if sub.SubmittedAt == nil || *sub.SubmittedAt != "2026-06-10T23:55:00Z" {
		t.Errorf("sub.SubmittedAt = %v, want 2026-06-10T23:55:00Z", sub.SubmittedAt)
	}
	if sub.WorkflowState != "submitted" {
		t.Errorf("sub.WorkflowState = %q, want %q", sub.WorkflowState, "submitted")
	}
	if sub.Late {
		t.Error("sub.Late should be false")
	}
	if sub.Attempt == nil || *sub.Attempt != 2 {
		t.Errorf("sub.Attempt = %v, want 2", sub.Attempt)
	}
}

func TestGetSubmissionLate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		submittedAt := "2026-06-12T01:30:00Z"
		score := 70.0
		json.NewEncoder(w).Encode(Submission{
			ID:            "502",
			UserID:        "790",
			AssignmentID:  "301",
			Score:         &score,
			SubmittedAt:   &submittedAt,
			WorkflowState: "submitted",
			Late:          true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	sub, err := GetSubmission(context.Background(), c, "42", "301", "790")
	if err != nil {
		t.Fatalf("GetSubmission() error: %v", err)
	}

	if !sub.Late {
		t.Error("sub.Late should be true")
	}
	if sub.Score == nil || *sub.Score != 70.0 {
		t.Errorf("sub.Score = %v, want 70.0", sub.Score)
	}
}

func TestSubmitAssignmentText(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)

		// Decode the request body
		json.NewDecoder(r.Body).Decode(&gotBody)

		submittedAt := "2026-06-11T14:30:00Z"
		json.NewEncoder(w).Encode(Submission{
			ID:            "503",
			UserID:        "789",
			AssignmentID:  "301",
			SubmittedAt:   &submittedAt,
			WorkflowState: "submitted",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	sub, err := SubmitAssignment(context.Background(), c, "42", "301", SubmissionRequest{
		SubmissionType: "online_text_entry",
		Body:           "<p>My essay text</p>",
	})
	if err != nil {
		t.Fatalf("SubmitAssignment() error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	wantPath := "/api/v1/courses/42/assignments/301/submissions"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody["submission"]["submission_type"] != "online_text_entry" {
		t.Errorf("submission_type = %q, want online_text_entry", gotBody["submission"]["submission_type"])
	}
	if gotBody["submission"]["body"] != "<p>My essay text</p>" {
		t.Errorf("body = %q, want <p>My essay text</p>", gotBody["submission"]["body"])
	}
	if sub.ID != "503" {
		t.Errorf("sub.ID = %q, want %q", sub.ID, "503")
	}
	if sub.WorkflowState != "submitted" {
		t.Errorf("sub.WorkflowState = %q, want submitted", sub.WorkflowState)
	}
}

func TestSubmitAssignmentURL(t *testing.T) {
	var gotBody map[string]map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewDecoder(r.Body).Decode(&gotBody)

		submittedAt := "2026-06-11T15:00:00Z"
		json.NewEncoder(w).Encode(Submission{
			ID:            "504",
			UserID:        "789",
			AssignmentID:  "301",
			SubmittedAt:   &submittedAt,
			WorkflowState: "submitted",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	sub, err := SubmitAssignment(context.Background(), c, "42", "301", SubmissionRequest{
		SubmissionType: "online_url",
		URL:            "https://example.com/my-project",
	})
	if err != nil {
		t.Fatalf("SubmitAssignment() error: %v", err)
	}

	if gotBody["submission"]["submission_type"] != "online_url" {
		t.Errorf("submission_type = %q, want online_url", gotBody["submission"]["submission_type"])
	}
	if gotBody["submission"]["url"] != "https://example.com/my-project" {
		t.Errorf("url = %q, want https://example.com/my-project", gotBody["submission"]["url"])
	}
	if sub.ID != "504" {
		t.Errorf("sub.ID = %q, want %q", sub.ID, "504")
	}
}

func TestSubmitAssignmentFile(t *testing.T) {
	var gotBody map[string]map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewDecoder(r.Body).Decode(&gotBody)

		submittedAt := "2026-06-11T15:30:00Z"
		json.NewEncoder(w).Encode(Submission{
			ID:            "505",
			UserID:        "789",
			AssignmentID:  "301",
			SubmittedAt:   &submittedAt,
			WorkflowState: "submitted",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	sub, err := SubmitAssignment(context.Background(), c, "42", "301", SubmissionRequest{
		SubmissionType: "online_upload",
		FileIDs:        []string{"1001", "1002"},
	})
	if err != nil {
		t.Fatalf("SubmitAssignment() error: %v", err)
	}

	if gotBody["submission"]["submission_type"] != "online_upload" {
		t.Errorf("submission_type = %q, want online_upload", gotBody["submission"]["submission_type"])
	}
	// Verify file_ids were sent as an array
	fileIDs, ok := gotBody["submission"]["file_ids"].([]any)
	if !ok {
		t.Fatalf("file_ids type = %T, want []any", gotBody["submission"]["file_ids"])
	}
	if len(fileIDs) != 2 {
		t.Errorf("len(file_ids) = %d, want 2", len(fileIDs))
	}
	if sub.ID != "505" {
		t.Errorf("sub.ID = %q, want %q", sub.ID, "505")
	}
}

func TestSubmitAssignmentInvalidType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not reach here
		t.Error("server should not have been called for invalid submission type")
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := SubmitAssignment(context.Background(), c, "42", "301", SubmissionRequest{
		SubmissionType: "online_quiz",
	})
	if err == nil {
		t.Fatal("expected error for invalid submission type, got nil")
	}
}

func TestListSubmissions(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		score := 88.0
		grade := "B+"
		submittedAt := "2026-06-10T23:55:00Z"
		json.NewEncoder(w).Encode([]Submission{
			{
				ID:           "501",
				UserID:       "789",
				AssignmentID: "301",
				Score:        &score,
				Grade:        &grade,
				SubmittedAt:  &submittedAt,
				User:         &User{ID: "789", Name: "Alice Smith", SortableName: "Smith, Alice"},
			},
			{
				ID:           "502",
				UserID:       "790",
				AssignmentID: "301",
				User:         &User{ID: "790", Name: "Bob Jones", SortableName: "Jones, Bob"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	submissions, meta, err := ListSubmissions(context.Background(), c, "42", "301", RequestOptions{})
	if err != nil {
		t.Fatalf("ListSubmissions() error: %v", err)
	}

	if gotPath != "/api/v1/courses/42/assignments/301/submissions" {
		t.Errorf("path = %q, want %q", gotPath, "/api/v1/courses/42/assignments/301/submissions")
	}

	parsed, _ := url.ParseQuery(gotQuery)
	if parsed.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", parsed.Get("per_page"), "100")
	}
	if parsed.Get("include[]") != "user" {
		t.Errorf("include[] = %q, want %q", parsed.Get("include[]"), "user")
	}

	if len(submissions) != 2 {
		t.Fatalf("len(submissions) = %d, want 2", len(submissions))
	}
	if submissions[0].ID != "501" {
		t.Errorf("submissions[0].ID = %q, want %q", submissions[0].ID, "501")
	}
	if submissions[0].User == nil {
		t.Fatal("submissions[0].User should not be nil")
	}
	if submissions[0].User.Name != "Alice Smith" {
		t.Errorf("submissions[0].User.Name = %q, want %q", submissions[0].User.Name, "Alice Smith")
	}
	if submissions[1].User == nil {
		t.Fatal("submissions[1].User should not be nil")
	}
	if submissions[1].User.Name != "Bob Jones" {
		t.Errorf("submissions[1].User.Name = %q, want %q", submissions[1].User.Name, "Bob Jones")
	}

	if !meta.Paginated {
		t.Error("meta.Paginated should be true")
	}
}

func TestListSubmissionsPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/42/assignments/301/submissions?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]Submission{
				{ID: "501", UserID: "789", User: &User{ID: "789", Name: "Alice"}},
			})
		case 2:
			json.NewEncoder(w).Encode([]Submission{
				{ID: "502", UserID: "790", User: &User{ID: "790", Name: "Bob"}},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	submissions, meta, err := ListSubmissions(context.Background(), c, "42", "301", RequestOptions{})
	if err != nil {
		t.Fatalf("ListSubmissions() error: %v", err)
	}

	if len(submissions) != 2 {
		t.Fatalf("len(submissions) = %d, want 2", len(submissions))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
}

func TestListSubmissionsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	_, _, err := ListSubmissions(context.Background(), c, "42", "301", RequestOptions{})
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}
