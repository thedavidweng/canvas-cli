package canvas

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConversationList(t *testing.T) {
	var gotPath string
	var gotPerPage string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Conversation{
			{
				ID:            "1001",
				Subject:       "Homework question",
				WorkflowState: "unread",
				LastMessage:   "Can you help with problem 3?",
				LastMessageAt: "2026-06-11T14:30:00Z",
				MessageCount:  2,
				Participants: []User{
					{ID: "10", Name: "Alice"},
					{ID: "20", Name: "Bob"},
				},
			},
			{
				ID:            "1002",
				Subject:       "Project update",
				WorkflowState: "read",
				LastMessage:   "The deadline is next Friday.",
				LastMessageAt: "2026-06-12T09:00:00Z",
				MessageCount:  5,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	conversations, meta, err := ListConversations(context.Background(), c, RequestOptions{PageSize: 100})
	if err != nil {
		t.Fatalf("ListConversations() error: %v", err)
	}

	wantPath := "/api/v1/conversations"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotPerPage != "100" {
		t.Errorf("per_page = %q, want %q", gotPerPage, "100")
	}

	if len(conversations) != 2 {
		t.Fatalf("len(conversations) = %d, want 2", len(conversations))
	}
	if conversations[0].ID != "1001" {
		t.Errorf("conversations[0].ID = %q, want %q", conversations[0].ID, "1001")
	}
	if conversations[0].Subject != "Homework question" {
		t.Errorf("conversations[0].Subject = %q, want %q", conversations[0].Subject, "Homework question")
	}
	if conversations[0].WorkflowState != "unread" {
		t.Errorf("conversations[0].WorkflowState = %q, want %q", conversations[0].WorkflowState, "unread")
	}
	if conversations[0].LastMessage != "Can you help with problem 3?" {
		t.Errorf("conversations[0].LastMessage = %q, want %q", conversations[0].LastMessage, "Can you help with problem 3?")
	}
	if len(conversations[0].Participants) != 2 {
		t.Errorf("len(conversations[0].Participants) = %d, want 2", len(conversations[0].Participants))
	}
	if conversations[1].ID != "1002" {
		t.Errorf("conversations[1].ID = %q, want %q", conversations[1].ID, "1002")
	}
	if conversations[1].WorkflowState != "read" {
		t.Errorf("conversations[1].WorkflowState = %q, want %q", conversations[1].WorkflowState, "read")
	}
	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
}

func TestConversationGet(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Conversation{
			ID:            "1001",
			Subject:       "Homework question",
			WorkflowState: "read",
			LastMessage:   "Thanks for the help!",
			LastMessageAt: "2026-06-12T10:00:00Z",
			MessageCount:  3,
			Participants: []User{
				{ID: "10", Name: "Alice"},
				{ID: "20", Name: "Bob"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	conversation, err := GetConversation(context.Background(), c, "1001")
	if err != nil {
		t.Fatalf("GetConversation() error: %v", err)
	}

	wantPath := "/api/v1/conversations/1001"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if conversation.ID != "1001" {
		t.Errorf("conversation.ID = %q, want %q", conversation.ID, "1001")
	}
	if conversation.Subject != "Homework question" {
		t.Errorf("conversation.Subject = %q, want %q", conversation.Subject, "Homework question")
	}
	if conversation.WorkflowState != "read" {
		t.Errorf("conversation.WorkflowState = %q, want %q", conversation.WorkflowState, "read")
	}
	if conversation.LastMessage != "Thanks for the help!" {
		t.Errorf("conversation.LastMessage = %q, want %q", conversation.LastMessage, "Thanks for the help!")
	}
	if conversation.MessageCount != 3 {
		t.Errorf("conversation.MessageCount = %d, want 3", conversation.MessageCount)
	}
	if len(conversation.Participants) != 2 {
		t.Errorf("len(conversation.Participants) = %d, want 2", len(conversation.Participants))
	}
}

func TestConversationSendMessage(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		json.Unmarshal(bodyBytes, &gotBody)

		json.NewEncoder(w).Encode(Conversation{
			ID:            "1003",
			Subject:       "New assignment question",
			WorkflowState: "unread",
			LastMessage:   "When is the deadline?",
			MessageCount:  1,
			Participants: []User{
				{ID: "10", Name: "Alice"},
				{ID: "30", Name: "Charlie"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	conversation, err := SendMessage(context.Background(), c, []string{"30", "40"}, "New assignment question", "When is the deadline?")
	if err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	wantPath := "/api/v1/conversations"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want %q", gotMethod, "POST")
	}

	recipients, ok := gotBody["recipients"].([]any)
	if !ok {
		t.Fatalf("recipients type = %T, want []any", gotBody["recipients"])
	}
	if len(recipients) != 2 {
		t.Errorf("len(recipients) = %d, want 2", len(recipients))
	}
	if recipients[0].(string) != "30" {
		t.Errorf("recipients[0] = %q, want %q", recipients[0], "30")
	}
	if recipients[1].(string) != "40" {
		t.Errorf("recipients[1] = %q, want %q", recipients[1], "40")
	}
	if gotBody["subject"] != "New assignment question" {
		t.Errorf("subject = %q, want %q", gotBody["subject"], "New assignment question")
	}
	if gotBody["body"] != "When is the deadline?" {
		t.Errorf("body = %q, want %q", gotBody["body"], "When is the deadline?")
	}

	if conversation.ID != "1003" {
		t.Errorf("conversation.ID = %q, want %q", conversation.ID, "1003")
	}
	if conversation.Subject != "New assignment question" {
		t.Errorf("conversation.Subject = %q, want %q", conversation.Subject, "New assignment question")
	}
	if conversation.WorkflowState != "unread" {
		t.Errorf("conversation.WorkflowState = %q, want %q", conversation.WorkflowState, "unread")
	}
	if len(conversation.Participants) != 2 {
		t.Errorf("len(conversation.Participants) = %d, want 2", len(conversation.Participants))
	}
}

func TestConversationReply(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		json.Unmarshal(bodyBytes, &gotBody)

		json.NewEncoder(w).Encode(Conversation{
			ID:            "1001",
			Subject:       "Homework question",
			WorkflowState: "read",
			LastMessage:   "The deadline is Friday.",
			MessageCount:  4,
			Participants: []User{
				{ID: "10", Name: "Alice"},
				{ID: "20", Name: "Bob"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	conversation, err := ReplyToConversation(context.Background(), c, "1001", "The deadline is Friday.")
	if err != nil {
		t.Fatalf("ReplyToConversation() error: %v", err)
	}

	wantPath := "/api/v1/conversations/1001/add_message"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want %q", gotMethod, "POST")
	}
	if gotBody["body"] != "The deadline is Friday." {
		t.Errorf("body = %q, want %q", gotBody["body"], "The deadline is Friday.")
	}

	if conversation.ID != "1001" {
		t.Errorf("conversation.ID = %q, want %q", conversation.ID, "1001")
	}
	if conversation.MessageCount != 4 {
		t.Errorf("conversation.MessageCount = %d, want 4", conversation.MessageCount)
	}
	if conversation.LastMessage != "The deadline is Friday." {
		t.Errorf("conversation.LastMessage = %q, want %q", conversation.LastMessage, "The deadline is Friday.")
	}
}

func TestConversationArchive(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		json.Unmarshal(bodyBytes, &gotBody)

		json.NewEncoder(w).Encode(Conversation{
			ID:            "1001",
			Subject:       "Homework question",
			WorkflowState: "archived",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	err := ArchiveConversation(context.Background(), c, "1001")
	if err != nil {
		t.Fatalf("ArchiveConversation() error: %v", err)
	}

	wantPath := "/api/v1/conversations/1001"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotMethod != "PUT" {
		t.Errorf("method = %q, want %q", gotMethod, "PUT")
	}
	if gotBody["workflow_state"] != "archived" {
		t.Errorf("workflow_state = %q, want %q", gotBody["workflow_state"], "archived")
	}
}

func TestListConversationsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, _, err := ListConversations(context.Background(), c, RequestOptions{})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestGetConversationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GetConversation(context.Background(), c, "9999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestSendMessageError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := SendMessage(context.Background(), c, []string{"30"}, "Subject", "Body")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestReplyToConversationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := ReplyToConversation(context.Background(), c, "1001", "Hello")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestArchiveConversationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	err := ArchiveConversation(context.Background(), c, "1001")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}
