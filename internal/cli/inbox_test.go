package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestInboxCmd_Exists(t *testing.T) {
	cmd := NewInboxCmd()
	if cmd.Use != "inbox" {
		t.Errorf("expected Use 'inbox', got %q", cmd.Use)
	}
}

func TestInboxCmd_HasSubcommands(t *testing.T) {
	cmd := NewInboxCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range []string{"list", "get", "send", "reply", "archive"} {
		if !subs[want] {
			t.Errorf("expected '%s' subcommand", want)
		}
	}
}

func TestInboxList_JSONReturnsConversations(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations", 200, []map[string]any{
		{
			"id":             "1001",
			"subject":        "Homework question",
			"workflow_state": "unread",
			"last_message":   "Can you help?",
			"message_count":  2,
		},
		{
			"id":             "1002",
			"subject":        "Project update",
			"workflow_state": "read",
			"last_message":   "Deadline is Friday.",
			"message_count":  5,
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("inbox list --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var conversations []canvas.Conversation
	if err := json.Unmarshal(dataJSON, &conversations); err != nil {
		t.Fatalf("data is not []Conversation: %v", err)
	}
	if len(conversations) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(conversations))
	}
	if conversations[0].Subject != "Homework question" {
		t.Errorf("expected subject 'Homework question', got %q", conversations[0].Subject)
	}
}

func TestInboxGet_JSONReturnsConversation(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations/1001", 200, map[string]any{
		"id":             "1001",
		"subject":        "Homework question",
		"workflow_state": "read",
		"last_message":   "Thanks for the help!",
		"message_count":  3,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"1001"})
	if err != nil {
		t.Fatalf("inbox get --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var conversation canvas.Conversation
	if err := json.Unmarshal(dataJSON, &conversation); err != nil {
		t.Fatalf("data is not Conversation: %v", err)
	}
	if conversation.ID != "1001" {
		t.Errorf("expected ID '1001', got %q", conversation.ID)
	}
	if conversation.Subject != "Homework question" {
		t.Errorf("expected subject 'Homework question', got %q", conversation.Subject)
	}
}

func TestInboxSend_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxSendCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("to", "42")
	_ = cmd.Flags().Set("subject", "Hello!")
	_ = cmd.Flags().Set("body", "How are you?")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("inbox send --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/conversations") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "Hello!") {
		t.Errorf("expected subject in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "How are you?") {
		t.Errorf("expected body in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestInboxSend_ConfirmSendsPOST(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/conversations", 200, map[string]any{
		"id":             "1003",
		"subject":        "Hello!",
		"workflow_state": "unread",
		"message_count":  1,
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newInboxSendCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("to", "42")
	_ = cmd.Flags().Set("subject", "Hello!")
	_ = cmd.Flags().Set("body", "How are you?")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("inbox send --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/conversations" {
		t.Errorf("expected path /api/v1/conversations, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "Hello!") {
		t.Errorf("expected subject in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "Message sent") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestInboxReply_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("body", "Thanks for the info!")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"1001"})
	if err != nil {
		t.Fatalf("inbox reply --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/conversations/1001/add_message") {
		t.Errorf("expected reply endpoint in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "Thanks for the info!") {
		t.Errorf("expected body in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestInboxReply_ConfirmSendsPOST(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/conversations/1001/add_message", 200, map[string]any{
		"id":             "1001",
		"subject":        "Homework question",
		"workflow_state": "read",
		"last_message":   "Thanks for the info!",
		"message_count":  4,
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newInboxReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("body", "Thanks for the info!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"1001"})
	if err != nil {
		t.Fatalf("inbox reply --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/conversations/1001/add_message" {
		t.Errorf("expected path /api/v1/conversations/1001/add_message, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "Thanks for the info!") {
		t.Errorf("expected body in request, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "Reply sent") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestInboxArchive_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxArchiveCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"1001"})
	if err != nil {
		t.Fatalf("inbox archive --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/conversations/1001") {
		t.Errorf("expected endpoint in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "archived") {
		t.Errorf("expected 'archived' in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestInboxCommands_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	commands := []struct {
		name string
		fn   func() *cobra.Command
		args []string
	}{
		{"send", newInboxSendCmd, nil},
		{"reply", newInboxReplyCmd, []string{"1001"}},
		{"archive", newInboxArchiveCmd, []string{"1001"}},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := tc.fn()
			cmd.SetContext(WithConfig(context.Background(), cfg))
			cmd.SetOut(&buf)

			// Set required flags for send
			if tc.name == "send" {
				_ = cmd.Flags().Set("to", "42")
				_ = cmd.Flags().Set("subject", "test")
				_ = cmd.Flags().Set("body", "test")
			}
			if tc.name == "reply" {
				_ = cmd.Flags().Set("body", "test")
			}

			err := cmd.RunE(cmd, tc.args)
			if err == nil {
				t.Fatalf("expected error in read-only mode for %s", tc.name)
			}
			if exitErr, ok := err.(interface{ ExitCode() int }); ok {
				if exitErr.ExitCode() != 7 {
					t.Errorf("expected exit code 7 for %s, got %d", tc.name, exitErr.ExitCode())
				}
			} else {
				t.Errorf("expected exit error with ExitCode() for %s, got %T", tc.name, err)
			}
		})
	}
}

func TestInboxCommands_WriteAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/conversations", 200, map[string]any{
		"id": "1003", "subject": "Hello!", "workflow_state": "unread", "message_count": 1,
	})
	mock.On("POST", "/api/v1/conversations/1001/add_message", 200, map[string]any{
		"id": "1001", "subject": "Re: Hello!", "workflow_state": "read", "message_count": 2,
	})
	mock.On("PUT", "/api/v1/conversations/1001", 200, map[string]any{
		"id": "1001", "subject": "Hello!", "workflow_state": "archived",
	})

	tests := []struct {
		name    string
		fn      func() *cobra.Command
		args    []string
		setup   func(*cobra.Command)
		wantCmd string
	}{
		{
			name:    "send",
			fn:      newInboxSendCmd,
			wantCmd: "inbox.send",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("to", "42")
				_ = cmd.Flags().Set("subject", "Hello!")
				_ = cmd.Flags().Set("body", "How are you?")
				_ = cmd.Flags().Set("confirm", "true")
			},
		},
		{
			name:    "reply",
			fn:      newInboxReplyCmd,
			args:    []string{"1001"},
			wantCmd: "inbox.reply",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("body", "Thanks!")
				_ = cmd.Flags().Set("confirm", "true")
			},
		},
		{
			name:    "archive",
			fn:      newInboxArchiveCmd,
			args:    []string{"1001"},
			wantCmd: "inbox.archive",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("confirm", "true")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auditPath := filepath.Join(t.TempDir(), "audit.jsonl")

			cfg := &config.ResolvedConfig{
				BaseURL:      mock.URL(),
				Token:        "test-token",
				Profile:      "default",
				AuditEnabled: true,
				AuditPath:    auditPath,
			}

			var buf bytes.Buffer
			cmd := tc.fn()
			cmd.SetContext(WithConfig(context.Background(), cfg))
			cmd.SetOut(&buf)
			if tc.setup != nil {
				tc.setup(cmd)
			}

			err := cmd.RunE(cmd, tc.args)
			if err != nil {
				t.Fatalf("inbox %s --confirm failed: %v", tc.name, err)
			}

			data, err := os.ReadFile(auditPath)
			if err != nil {
				t.Fatalf("failed to read audit log for %s: %v", tc.name, err)
			}
			if len(data) == 0 {
				t.Fatalf("audit log is empty for %s", tc.name)
			}
			if !strings.Contains(string(data), tc.wantCmd) {
				t.Errorf("expected '%s' in audit log, got: %s", tc.wantCmd, string(data))
			}
		})
	}
}

// --- inbox reply uncovered paths ---

func TestInboxList_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations", 200, []map[string]any{
		{"id": "1001", "subject": "Homework question", "workflow_state": "unread", "message_count": 2},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("inbox list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Homework question") {
		t.Errorf("expected 'Homework question' in output, got: %s", output)
	}
	if !strings.Contains(output, "1001") {
		t.Errorf("expected conversation ID in output, got: %s", output)
	}
}

func TestInboxList_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

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

func TestInboxList_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestInboxGet_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations/1001", 200, map[string]any{
		"id":             "1001",
		"subject":        "Homework question",
		"workflow_state": "read",
		"last_message":   "Thanks for the help!",
		"message_count":  3,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"1001"})
	if err != nil {
		t.Fatalf("inbox get failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Homework question") {
		t.Errorf("expected subject in output, got: %s", output)
	}
	if !strings.Contains(output, "Thanks for the help!") {
		t.Errorf("expected last message in output, got: %s", output)
	}
	if !strings.Contains(output, "3") {
		t.Errorf("expected message count in output, got: %s", output)
	}
}

func TestInboxGet_APIError_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations/1001", 500, map[string]any{
		"errors": []map[string]any{{"message": "not found"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"1001"})
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

func TestInboxGet_APIError_Human(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/conversations/1001", 500, map[string]any{
		"errors": []map[string]any{{"message": "not found"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"1001"})
	if err == nil {
		t.Fatal("expected error in human mode")
	}
}

func TestInboxReply_MissingBody(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	// body not set (empty by default)

	err := cmd.RunE(cmd, []string{"1001"})
	if err == nil {
		t.Fatal("expected error when --body is missing, got nil")
	}
	if !strings.Contains(err.Error(), "--body is required") {
		t.Errorf("expected '--body is required' in error, got: %v", err)
	}
}

func TestInboxReply_NilConfig(t *testing.T) {
	var buf bytes.Buffer
	cmd := newInboxReplyCmd()
	cmd.SetContext(context.Background()) // no config
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("body", "test")

	err := cmd.RunE(cmd, []string{"1001"})
	if err == nil {
		t.Fatal("expected error when config is nil, got nil")
	}
	if !strings.Contains(err.Error(), "no config loaded") {
		t.Errorf("expected 'no config loaded' in error, got: %v", err)
	}
}

func TestInboxReply_APIError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/conversations/1001/add_message", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newInboxReplyCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("body", "Hello!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"1001"})
	if err == nil {
		t.Fatal("expected error on API failure, got nil")
	}
}
