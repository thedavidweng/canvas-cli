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

func TestAnnouncementsCmd_Exists(t *testing.T) {
	cmd := NewAnnouncementsCmd()
	if cmd.Use != "announcements" {
		t.Errorf("expected Use 'announcements', got %q", cmd.Use)
	}
}

func TestAnnouncementsCmd_HasListSubcommand(t *testing.T) {
	cmd := NewAnnouncementsCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'list' subcommand")
	}
}

func TestAnnouncementsList_ReturnsDiscussionTopics(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/announcements", 200, []map[string]any{
		{
			"id":              "100",
			"title":           "Welcome to CS101",
			"message":         "Welcome everyone!",
			"is_announcement": true,
			"published":       true,
			"discussion_type": "side_comment",
		},
		{
			"id":              "101",
			"title":           "Midterm Info",
			"message":         "Midterm is next week.",
			"is_announcement": true,
			"published":       true,
			"discussion_type": "side_comment",
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAnnouncementsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("announcements list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Welcome to CS101") {
		t.Errorf("expected 'Welcome to CS101' in output, got: %s", output)
	}
	if !strings.Contains(output, "Midterm Info") {
		t.Errorf("expected 'Midterm Info' in output, got: %s", output)
	}

	// Verify the request hit the announcements endpoint with context_codes
	last := mock.LastRequest()
	if last.Path != "/api/v1/announcements" {
		t.Errorf("expected request to /api/v1/announcements, got %s", last.Path)
	}
	ctxCodes := last.Query.Get("context_codes[]")
	if ctxCodes != "course_1" {
		t.Errorf("expected context_codes[]=course_1, got %q", ctxCodes)
	}
}

func TestAnnouncementsList_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/announcements", 200, []map[string]any{
		{
			"id":              "100",
			"title":           "Welcome to CS101",
			"message":         "Welcome everyone!",
			"is_announcement": true,
			"published":       true,
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAnnouncementsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("announcements list --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true in envelope")
	}
	if env.Data == nil {
		t.Fatal("expected data in envelope")
	}

	dataJSON, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("failed to re-marshal data: %v", err)
	}
	var announcements []canvas.DiscussionTopic
	if err := json.Unmarshal(dataJSON, &announcements); err != nil {
		t.Fatalf("data is not []DiscussionTopic: %v", err)
	}
	if len(announcements) != 1 {
		t.Errorf("expected 1 announcement, got %d", len(announcements))
	}
	if announcements[0].Title != "Welcome to CS101" {
		t.Errorf("expected title 'Welcome to CS101', got %q", announcements[0].Title)
	}
}

func TestAnnouncementsCreate_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Write a temp body file
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte("Welcome to the course!"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAnnouncementsCreateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("title", "Welcome")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("announcements create --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/discussion_topics") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "Welcome") {
		t.Errorf("expected title in dry-run output, got: %s", output)
	}
	// Verify no actual request was made
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestAnnouncementsCreate_ConfirmSendsPOST(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics", 200, map[string]any{
		"id":              "500",
		"title":           "Welcome",
		"message":         "Welcome to the course!",
		"is_announcement": true,
		"published":       true,
	})

	// Write a temp body file
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte("Welcome to the course!"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newAnnouncementsCreateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("title", "Welcome")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("announcements create --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/discussion_topics" {
		t.Errorf("expected path /api/v1/courses/1/discussion_topics, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "Welcome to the course!") {
		t.Errorf("expected body in request, got: %s", last.Body)
	}
	if !strings.Contains(last.Body, `"is_announcement":true`) && !strings.Contains(last.Body, `"is_announcement": true`) {
		t.Errorf("expected is_announcement=true in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "created") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestAnnouncementsCreate_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newAnnouncementsCreateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("title", "Welcome")
	_ = cmd.Flags().Set("body-file", "/dev/null")
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

func TestAnnouncementsCreate_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/discussion_topics", 200, map[string]any{
		"id":              "500",
		"title":           "Welcome",
		"message":         "Welcome to the course!",
		"is_announcement": true,
		"published":       true,
	})

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte("Welcome to the course!"), 0o644); err != nil {
		t.Fatal(err)
	}

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
	cmd := newAnnouncementsCreateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("title", "Welcome")
	_ = cmd.Flags().Set("body-file", bodyFile)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("announcements create --confirm failed: %v", err)
	}

	// Verify audit log was written
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "announcements.create") {
		t.Errorf("expected 'announcements.create' in audit log, got: %s", string(data))
	}
}
