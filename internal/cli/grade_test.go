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

func TestGradeCmd_Exists(t *testing.T) {
	cmd := NewGradingCmd()
	if cmd.Use != "grade" {
		t.Errorf("expected Use 'grade', got %q", cmd.Use)
	}
}

func TestGradeCmd_HasSubcommands(t *testing.T) {
	cmd := NewGradingCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range []string{"set", "comment", "import"} {
		if !subs[want] {
			t.Errorf("expected '%s' subcommand", want)
		}
	}
}

// --- grade set ---

func TestGradeSet_DryRunShowsPreviewNoRequest(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newGradeSetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("score", "95")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade set --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/assignments/10/submissions/42") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "95") {
		t.Errorf("expected score '95' in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestGradeSet_ConfirmSendsPUT(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id":             "500",
		"user_id":        "42",
		"assignment_id":  "10",
		"workflow_state": "graded",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeSetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("score", "95")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade set --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "PUT" {
		t.Errorf("expected PUT method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/assignments/10/submissions/42" {
		t.Errorf("expected path /api/v1/courses/1/assignments/10/submissions/42, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "95") {
		t.Errorf("expected score in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "Grade set") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestGradeSet_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newGradeSetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("score", "95")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in read-only mode")
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		if exitErr.ExitCode() != 7 {
			t.Errorf("expected exit code 7, got %d", exitErr.ExitCode())
		}
	} else {
		t.Errorf("expected exit error with ExitCode(), got %T", err)
	}
}

func TestGradeSet_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id":             "500",
		"user_id":        "42",
		"assignment_id":  "10",
		"workflow_state": "graded",
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
	cmd := newGradeSetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("score", "95")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade set --confirm failed: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "grade.set") {
		t.Errorf("expected 'grade.set' in audit log, got: %s", string(data))
	}
}

// --- grade comment ---

func TestGradeComment_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newGradeCommentCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("comment", "Good work!")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade comment --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/assignments/10/submissions/42") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "Good work!") {
		t.Errorf("expected comment text in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestGradeComment_ConfirmSendsPUT(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id":             "500",
		"user_id":        "42",
		"assignment_id":  "10",
		"workflow_state": "graded",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeCommentCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("comment", "Good work!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade comment --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "PUT" {
		t.Errorf("expected PUT method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/assignments/10/submissions/42" {
		t.Errorf("expected path /api/v1/courses/1/assignments/10/submissions/42, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "Good work!") {
		t.Errorf("expected comment in request body, got: %s", last.Body)
	}
}

func TestGradeComment_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newGradeCommentCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("comment", "Good work!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error in read-only mode")
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		if exitErr.ExitCode() != 7 {
			t.Errorf("expected exit code 7, got %d", exitErr.ExitCode())
		}
	} else {
		t.Errorf("expected exit error with ExitCode(), got %T", err)
	}
}

func TestGradeComment_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id":             "500",
		"user_id":        "42",
		"assignment_id":  "10",
		"workflow_state": "graded",
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
	cmd := newGradeCommentCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("comment", "Good work!")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade comment --confirm failed: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "grade.comment") {
		t.Errorf("expected 'grade.comment' in audit log, got: %s", string(data))
	}
}

// --- grade import ---

func TestGradeImport_DryRunShowsPreviewWithAffectedCount(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "grades.csv")
	os.WriteFile(csvPath, []byte("user_id,score\n42,95\n43,88\n"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newGradeImportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("csv", csvPath)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade import --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/assignments/10/submissions/update_grades") {
		t.Errorf("expected bulk update endpoint in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "2") {
		t.Errorf("expected affected count '2' in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestGradeImport_ConfirmSendsPOST(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/assignments/10/submissions/update_grades", 200, []map[string]any{
		{"id": "500", "user_id": "42", "workflow_state": "graded"},
		{"id": "501", "user_id": "43", "workflow_state": "graded"},
	})

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "grades.csv")
	os.WriteFile(csvPath, []byte("user_id,score\n42,95\n43,88\n"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeImportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("csv", csvPath)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade import --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "POST" {
		t.Errorf("expected POST method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/assignments/10/submissions/update_grades" {
		t.Errorf("expected bulk update path, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "42") {
		t.Errorf("expected user 42 in request body, got: %s", last.Body)
	}
}

func TestGradeImport_ConfirmWithoutDryRunMayWarn(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/assignments/10/submissions/update_grades", 200, []map[string]any{
		{"id": "500", "user_id": "42", "workflow_state": "graded"},
	})

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "grades.csv")
	os.WriteFile(csvPath, []byte("user_id,score\n42,95\n"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeImportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("csv", csvPath)
	_ = cmd.Flags().Set("confirm", "true")
	// No --dry-run

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade import --confirm without --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "warning") {
		t.Errorf("expected warning about importing without dry-run, got: %s", output)
	}
}

func TestGradeImport_PartialFailureReturnsExit8(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Return a 500 for the bulk update to simulate partial failure
	mock.On("POST", "/api/v1/courses/1/assignments/10/submissions/update_grades", 500, map[string]any{
		"errors": []map[string]any{
			{"message": "internal server error"},
		},
	})

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "grades.csv")
	os.WriteFile(csvPath, []byte("user_id,score\n42,95\n43,88\n"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeImportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("csv", csvPath)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		if exitErr.ExitCode() != 8 {
			t.Errorf("expected exit code 8 for partial failure, got %d", exitErr.ExitCode())
		}
	} else {
		t.Errorf("expected exit error with ExitCode(), got %T", err)
	}
}

func TestGradeImport_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/assignments/10/submissions/update_grades", 200, []map[string]any{
		{"id": "500", "user_id": "42", "workflow_state": "graded"},
	})

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "grades.csv")
	os.WriteFile(csvPath, []byte("user_id,score\n42,95\n"), 0644)

	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	var buf bytes.Buffer
	cmd := newGradeImportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("csv", csvPath)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade import --confirm failed: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "grade.import") {
		t.Errorf("expected 'grade.import' in audit log, got: %s", string(data))
	}
}

// --- JSON envelope ---

func TestGradeSet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id":             "500",
		"user_id":        "42",
		"assignment_id":  "10",
		"workflow_state": "graded",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeSetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("score", "95")
	_ = cmd.Flags().Set("confirm", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade set --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var sub canvas.Submission
	if err := json.Unmarshal(dataJSON, &sub); err != nil {
		t.Fatalf("data is not Submission: %v", err)
	}
	if sub.ID != "500" {
		t.Errorf("expected submission ID '500', got %q", sub.ID)
	}
}

func TestGradeComment_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id":             "500",
		"user_id":        "42",
		"assignment_id":  "10",
		"workflow_state": "graded",
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeCommentCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")
	_ = cmd.Flags().Set("comment", "Nice work")
	_ = cmd.Flags().Set("confirm", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade comment --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

func TestGradeImport_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/assignments/10/submissions/update_grades", 200, []map[string]any{
		{"id": "500", "user_id": "42", "workflow_state": "graded"},
	})

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "grades.csv")
	os.WriteFile(csvPath, []byte("user_id,score\n42,95\n"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newGradeImportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("csv", csvPath)
	_ = cmd.Flags().Set("confirm", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("grade import --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

// --- All grade mutations write audit log ---

func TestGradeMutations_AllWriteAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id": "500", "user_id": "42", "workflow_state": "graded",
	})
	mock.On("POST", "/api/v1/courses/1/assignments/10/submissions/update_grades", 200, []map[string]any{
		{"id": "500", "user_id": "42", "workflow_state": "graded"},
	})

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "grades.csv")
	os.WriteFile(csvPath, []byte("user_id,score\n42,95\n"), 0644)

	tests := []struct {
		name    string
		fn      func() *cobra.Command
		setup   func(*cobra.Command)
		wantCmd string
	}{
		{
			name:    "set",
			fn:      newGradeSetCmd,
			wantCmd: "grade.set",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("course", "1")
				_ = cmd.Flags().Set("assignment", "10")
				_ = cmd.Flags().Set("user", "42")
				_ = cmd.Flags().Set("score", "95")
				_ = cmd.Flags().Set("confirm", "true")
			},
		},
		{
			name:    "comment",
			fn:      newGradeCommentCmd,
			wantCmd: "grade.comment",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("course", "1")
				_ = cmd.Flags().Set("assignment", "10")
				_ = cmd.Flags().Set("user", "42")
				_ = cmd.Flags().Set("comment", "Nice")
				_ = cmd.Flags().Set("confirm", "true")
			},
		},
		{
			name:    "import",
			fn:      newGradeImportCmd,
			wantCmd: "grade.import",
			setup: func(cmd *cobra.Command) {
				_ = cmd.Flags().Set("course", "1")
				_ = cmd.Flags().Set("assignment", "10")
				_ = cmd.Flags().Set("csv", csvPath)
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

			err := cmd.RunE(cmd, nil)
			if err != nil {
				t.Fatalf("grade %s --confirm failed: %v", tc.name, err)
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
