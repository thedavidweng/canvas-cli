package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestDiscussionsCmd_Exists(t *testing.T) {
	cmd := NewDiscussionsCmd()
	if cmd.Use != "discussions" {
		t.Errorf("expected Use 'discussions', got %q", cmd.Use)
	}
}

func TestDiscussionsCmd_HasSubcommands(t *testing.T) {
	cmd := NewDiscussionsCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range []string{"list", "get", "entries"} {
		if !subs[want] {
			t.Errorf("expected '%s' subcommand", want)
		}
	}
}

func TestDiscussionsList_ReturnsDiscussionTopics(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics", 200, []map[string]any{
		{
			"id":              "200",
			"title":           "Introductions",
			"message":         "Introduce yourself!",
			"discussion_type": "side_comment",
			"published":       true,
		},
		{
			"id":              "201",
			"title":           "Week 1 Discussion",
			"message":         "Discuss the readings.",
			"discussion_type": "side_comment",
			"published":       true,
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Introductions") {
		t.Errorf("expected 'Introductions' in output, got: %s", output)
	}
	if !strings.Contains(output, "Week 1 Discussion") {
		t.Errorf("expected 'Week 1 Discussion' in output, got: %s", output)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/discussion_topics" {
		t.Errorf("expected request to /api/v1/courses/1/discussion_topics, got %s", last.Path)
	}
}

func TestDiscussionsList_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics", 200, []map[string]any{
		{"id": "200", "title": "Introductions", "message": "Hi!", "discussion_type": "side_comment", "published": true},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions list --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var discussions []canvas.DiscussionTopic
	if err := json.Unmarshal(dataJSON, &discussions); err != nil {
		t.Fatalf("data is not []DiscussionTopic: %v", err)
	}
	if len(discussions) != 1 {
		t.Errorf("expected 1 discussion, got %d", len(discussions))
	}
	if discussions[0].Title != "Introductions" {
		t.Errorf("expected title 'Introductions', got %q", discussions[0].Title)
	}
}

func TestDiscussionsGet_ReturnsDiscussionTopic(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200", 200, map[string]any{
		"id":              "200",
		"title":           "Introductions",
		"message":         "Introduce yourself!",
		"discussion_type": "side_comment",
		"published":       true,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"200"})
	if err != nil {
		t.Fatalf("discussions get failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Introductions") {
		t.Errorf("expected 'Introductions' in output, got: %s", output)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/discussion_topics/200" {
		t.Errorf("expected request to /api/v1/courses/1/discussion_topics/200, got %s", last.Path)
	}
}

func TestDiscussionsGet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200", 200, map[string]any{
		"id":              "200",
		"title":           "Introductions",
		"message":         "Introduce yourself!",
		"discussion_type": "side_comment",
		"published":       true,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"200"})
	if err != nil {
		t.Fatalf("discussions get --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var topic canvas.DiscussionTopic
	if err := json.Unmarshal(dataJSON, &topic); err != nil {
		t.Fatalf("data is not DiscussionTopic: %v", err)
	}
	if topic.ID != "200" {
		t.Errorf("expected topic ID '200', got %q", topic.ID)
	}
}

func TestDiscussionsEntries_ReturnsEntries(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200/entries", 200, []map[string]any{
		{
			"id":         "300",
			"user_id":    "1",
			"user_name":  "Test User",
			"message":    "Hello everyone!",
			"created_at": "2026-01-01T00:00:00Z",
		},
		{
			"id":         "301",
			"user_id":    "2",
			"user_name":  "Another User",
			"message":    "Hi there!",
			"created_at": "2026-01-01T01:00:00Z",
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsEntriesCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"200"})
	if err != nil {
		t.Fatalf("discussions entries failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Hello everyone!") {
		t.Errorf("expected 'Hello everyone!' in output, got: %s", output)
	}
	if !strings.Contains(output, "Hi there!") {
		t.Errorf("expected 'Hi there!' in output, got: %s", output)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/discussion_topics/200/entries" {
		t.Errorf("expected request to /api/v1/courses/1/discussion_topics/200/entries, got %s", last.Path)
	}
}

func TestDiscussionsEntries_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200/entries", 200, []map[string]any{
		{"id": "300", "user_id": "1", "user_name": "Test User", "message": "Hello!"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsEntriesCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"200"})
	if err != nil {
		t.Fatalf("discussions entries --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var entries []canvas.DiscussionEntry
	if err := json.Unmarshal(dataJSON, &entries); err != nil {
		t.Fatalf("data is not []DiscussionEntry: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Message != "Hello!" {
		t.Errorf("expected message 'Hello!', got %q", entries[0].Message)
	}
}

func TestDiscussionsReply_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("message", "Great discussion!")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions reply --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/discussion_topics/200/entries") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "Great discussion!") {
		t.Errorf("expected message preview in dry-run output, got: %s", output)
	}
	// Verify no actual request was made
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestDiscussionsReply_ConfirmSendsPOST(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics/200/entries", 200, map[string]any{
		"id":       "901",
		"user_id":  "1",
		"message":  "Great discussion!",
		"username": "Test User",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("message", "Great discussion!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions reply --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/discussion_topics/200/entries" {
		t.Errorf("expected path /api/v1/courses/1/discussion_topics/200/entries, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "Great discussion!") {
		t.Errorf("expected message in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "Reply posted") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestDiscussionsReply_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("message", "test")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in read-only mode")
	}
	exitErr, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("expected exit error with ExitCode(), got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Errorf("expected exit code 7, got %d", exitErr.ExitCode())
	}
}

func TestDiscussionsReply_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics/200/entries", 200, map[string]any{
		"id":      "901",
		"user_id": "1",
		"message": "Great discussion!",
	})

	auditDir := t.TempDir()
	auditPath := filepath.Join(auditDir, "audit.jsonl")

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("message", "Great discussion!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions reply --confirm failed: %v", err)
	}

	// Verify audit log was written
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "discussions.reply") {
		t.Errorf("expected 'discussions.reply' in audit log, got: %s", string(data))
	}
}

func TestDiscussionsList_CourseRequired(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --course is missing")
	}
	if !strings.Contains(err.Error(), "--course") {
		t.Errorf("expected error about --course, got: %v", err)
	}
}

func TestDiscussionsList_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("expected no error in JSON mode, got: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false on API error")
	}
}

func TestDiscussionsList_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestDiscussionsGet_CourseRequired(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"200"})
	if err == nil {
		t.Fatal("expected error when --course is missing")
	}
	if !strings.Contains(err.Error(), "--course") {
		t.Errorf("expected error about --course, got: %v", err)
	}
}

func TestDiscussionsGet_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200", 500, map[string]any{
		"errors": []map[string]any{{"message": "not found"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"200"})
	if err != nil {
		t.Fatalf("expected no error in JSON mode, got: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false on API error")
	}
}

func TestDiscussionsGet_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200", 500, map[string]any{
		"errors": []map[string]any{{"message": "not found"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"200"})
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestDiscussionsEntries_CourseRequired(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsEntriesCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"200"})
	if err == nil {
		t.Fatal("expected error when --course is missing")
	}
	if !strings.Contains(err.Error(), "--course") {
		t.Errorf("expected error about --course, got: %v", err)
	}
}

func TestDiscussionsEntries_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200/entries", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsEntriesCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"200"})
	if err != nil {
		t.Fatalf("expected no error in JSON mode, got: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if env.OK {
		t.Error("expected ok:false on API error")
	}
}

func TestDiscussionsEntries_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200/entries", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsEntriesCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"200"})
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestDiscussionsEntries_HumanMode_EmptyUserName(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/discussion_topics/200/entries", 200, []map[string]any{
		{"id": "300", "user_id": "42", "message": "Hello!"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsEntriesCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"200"})
	if err != nil {
		t.Fatalf("discussions entries failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "42") {
		t.Errorf("expected user_id '42' in output, got: %s", output)
	}
	if !strings.Contains(output, "Hello!") {
		t.Errorf("expected 'Hello!' in output, got: %s", output)
	}
}

func TestDiscussionsReplyEntry_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("entry", "300")
	_ = cmd.Flags().Set("message", "I agree!")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions reply-entry --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/discussion_topics/200/entries/300/replies") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "I agree!") {
		t.Errorf("expected message in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestDiscussionsReplyEntry_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("entry", "300")
	_ = cmd.Flags().Set("message", "test")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in read-only mode")
	}
	exitErr, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("expected exit error with ExitCode(), got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Errorf("expected exit code 7, got %d", exitErr.ExitCode())
	}
}

func TestDiscussionsReplyEntry_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics/200/entries/300/replies", 200, map[string]any{
		"id":      "902",
		"user_id": "1",
		"message": "I agree!",
	})

	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("entry", "300")
	_ = cmd.Flags().Set("message", "I agree!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions reply-entry --confirm failed: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "discussions.reply-entry") {
		t.Errorf("expected 'discussions.reply-entry' in audit log, got: %s", string(data))
	}
}

func TestDiscussionsReplyEntry_MissingCourse(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --course is missing")
	}
	if !strings.Contains(err.Error(), "--course") {
		t.Errorf("expected error about --course, got: %v", err)
	}
}

func TestDiscussionsReplyEntry_MissingDid(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --did is missing")
	}
	if !strings.Contains(err.Error(), "--did") {
		t.Errorf("expected error about --did, got: %v", err)
	}
}

func TestDiscussionsReplyEntry_MissingEntry(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --entry is missing")
	}
	if !strings.Contains(err.Error(), "--entry") {
		t.Errorf("expected error about --entry, got: %v", err)
	}
}

func TestDiscussionsReplyEntry_MissingMessage(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("entry", "300")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --message is missing")
	}
	if !strings.Contains(err.Error(), "--message") {
		t.Errorf("expected error about --message, got: %v", err)
	}
}

func TestDiscussionsReplyEntry_Works(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics/200/entries/300/replies", 200, map[string]any{
		"id":         "902",
		"user_id":    "1",
		"message":    "I agree!",
		"parent_id":  "300",
		"created_at": "2026-06-12T12:00:00Z",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newDiscussionsReplyEntryCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("did", "200")
	_ = cmd.Flags().Set("entry", "300")
	_ = cmd.Flags().Set("message", "I agree!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions reply-entry --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/discussion_topics/200/entries/300/replies" {
		t.Errorf("expected reply path, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "I agree!") {
		t.Errorf("expected message in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "Reply posted") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

// --- discussions create ---

func TestDiscussionsCreate_ConfirmSendsPOST(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics", 200, map[string]any{
		"id":      "300",
		"title":   "New Discussion",
		"message": "Discuss the readings.",
	})

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("Discuss the readings."), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newDiscussionsCreateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("title", "New Discussion")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions create --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/discussion_topics" {
		t.Errorf("expected path /api/v1/courses/1/discussion_topics, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "New Discussion") {
		t.Errorf("expected title in request body, got: %s", last.Body)
	}
	if !strings.Contains(last.Body, "Discuss the readings.") {
		t.Errorf("expected message in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "created") {
		t.Errorf("expected success message in output, got: %s", output)
	}
	if !strings.Contains(output, "300") {
		t.Errorf("expected discussion ID '300' in output, got: %s", output)
	}
}

func TestDiscussionsCreate_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("Discuss the readings."), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newDiscussionsCreateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("title", "New Discussion")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("discussions create --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/discussion_topics") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "New Discussion") {
		t.Errorf("expected title in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestDiscussionsCreate_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(bodyFile, []byte("Discuss the readings."), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newDiscussionsCreateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("title", "New Discussion")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in read-only mode")
	}
	exitErr, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("expected exit error with ExitCode(), got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Errorf("expected exit code 7, got %d", exitErr.ExitCode())
	}
}
