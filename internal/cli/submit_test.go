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
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

func TestSubmit_DryRun_ShowsPreviewNoRequest(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in output, got: %s", output)
	}
	if !strings.Contains(output, "No mutation sent") {
		t.Errorf("expected 'No mutation sent' in output, got: %s", output)
	}

	// Should NOT have sent a POST request.
	for _, req := range mock.RequestLog() {
		if req.Method == "POST" {
			t.Errorf("dry-run should not send POST, got: %s %s", req.Method, req.Path)
		}
	}
}

func TestSubmit_Text_Confirm_SendsPost(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})
	mock.On("POST", "/api/v1/courses/1/assignments/100/submissions", 200, map[string]any{
		"id": "500", "workflow_state": "submitted", "assignment_id": "100",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify POST was sent.
	found := false
	for _, req := range mock.RequestLog() {
		if req.Method == "POST" && strings.HasSuffix(req.Path, "/submissions") {
			found = true
			if !strings.Contains(req.Body, `"online_text_entry"`) {
				t.Errorf("expected online_text_entry in request body, got: %s", req.Body)
			}
			if !strings.Contains(req.Body, `"my answer"`) {
				t.Errorf("expected body 'my answer' in request, got: %s", req.Body)
			}
			break
		}
	}
	if !found {
		t.Error("expected POST to /submissions, but none found")
	}
}

func TestSubmit_URL_Confirm_SendsPostWithURL(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Link Assignment", "course_id": "1",
		"submission_types": []string{"online_url"},
	})
	mock.On("POST", "/api/v1/courses/1/assignments/100/submissions", 200, map[string]any{
		"id": "501", "workflow_state": "submitted", "assignment_id": "100",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("url", "https://example.com/my-essay")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	found := false
	for _, req := range mock.RequestLog() {
		if req.Method == "POST" && strings.HasSuffix(req.Path, "/submissions") {
			found = true
			if !strings.Contains(req.Body, `"online_url"`) {
				t.Errorf("expected online_url in request body, got: %s", req.Body)
			}
			if !strings.Contains(req.Body, `"https://example.com/my-essay"`) {
				t.Errorf("expected URL in request body, got: %s", req.Body)
			}
			break
		}
	}
	if !found {
		t.Error("expected POST to /submissions, but none found")
	}
}

func TestSubmit_File_Confirm_UploadFlow(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// GET assignment
	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "File Assignment", "course_id": "1",
		"submission_types": []string{"online_upload"},
	})

	// Step 1: POST file upload initiation (returns upload_url and upload_params).
	mock.OnUploadInit("POST", "/api/v1/courses/1/files", "/upload/essay.pdf", map[string]string{
		"key": "value",
	})

	// Step 2: POST file content to upload_url (returns 201 with file JSON).
	mock.On("POST", "/upload/essay.pdf", 201, map[string]any{
		"id": "999", "filename": "essay.pdf", "display_name": "essay.pdf",
	})

	// POST submission.
	mock.On("POST", "/api/v1/courses/1/assignments/100/submissions", 200, map[string]any{
		"id": "502", "workflow_state": "submitted", "assignment_id": "100",
	})

	// Create a temp file.
	dir := t.TempDir()
	filePath := filepath.Join(dir, "essay.pdf")
	if err := os.WriteFile(filePath, []byte("fake pdf content"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("file", filePath)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify submission POST was sent with file_id.
	found := false
	for _, req := range mock.RequestLog() {
		if req.Method == "POST" && strings.HasSuffix(req.Path, "/submissions") {
			found = true
			if !strings.Contains(req.Body, `"online_upload"`) {
				t.Errorf("expected online_upload in request body, got: %s", req.Body)
			}
			if !strings.Contains(req.Body, `"999"`) {
				t.Errorf("expected file_id '999' in request body, got: %s", req.Body)
			}
			break
		}
	}
	if !found {
		t.Error("expected POST to /submissions, but none found")
	}
}

func TestSubmit_NoConfirm_InNonInteractiveMode_ReturnsError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	// Neither --confirm nor --dry-run set.

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err == nil {
		t.Fatal("expected error without --confirm, got nil")
	}

	safetyErr, ok := err.(*safety.SafetyError)
	if !ok {
		t.Fatalf("expected *safety.SafetyError, got %T: %v", err, err)
	}
	if safetyErr.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", safetyErr.ExitCode)
	}
	if !strings.Contains(safetyErr.Error(), "--confirm") {
		t.Errorf("expected error to mention --confirm, got: %s", safetyErr.Error())
	}
}

func TestSubmit_ReadOnly_ReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("read-only", "true")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err == nil {
		t.Fatal("expected error in read-only mode, got nil")
	}

	safetyErr, ok := err.(*safety.SafetyError)
	if !ok {
		t.Fatalf("expected *safety.SafetyError, got %T: %v", err, err)
	}
	if safetyErr.ExitCode != 7 {
		t.Errorf("expected exit code 7, got %d", safetyErr.ExitCode)
	}
	if !strings.Contains(safetyErr.Error(), "read-only") {
		t.Errorf("expected 'read-only' in error, got: %s", safetyErr.Error())
	}
}

func TestSubmit_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})
	mock.On("POST", "/api/v1/courses/1/assignments/100/submissions", 200, map[string]any{
		"id": "500", "workflow_state": "submitted", "assignment_id": "100",
	})

	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.jsonl")

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify audit file was written.
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	var event canvas.AuditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("unmarshal audit event: %v", err)
	}
	if event.Command != "assignments.submit" {
		t.Errorf("expected command 'assignments.submit', got %q", event.Command)
	}
	if event.Method != "POST" {
		t.Errorf("expected method 'POST', got %q", event.Method)
	}
	if !strings.Contains(event.Path, "/submissions") {
		t.Errorf("expected path to contain '/submissions', got %q", event.Path)
	}
	if event.DryRun {
		t.Error("expected DryRun=false for confirmed submission")
	}
	if !event.Success {
		t.Error("expected Success=true")
	}
	if event.Resource["course_id"] != "1" {
		t.Errorf("expected course_id '1', got %q", event.Resource["course_id"])
	}
	if event.Resource["assignment_id"] != "100" {
		t.Errorf("expected assignment_id '100', got %q", event.Resource["assignment_id"])
	}
}

func TestSubmit_RejectsEmptyBody(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")
	// No --text, --url, or --file set; no positional arg.

	err := cmd.RunE(cmd, []string{"100"})
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
	if !strings.Contains(err.Error(), "requires one of: --text, --url, or --file") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestSubmit_ValidatesSubmissionType(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Upload Assignment", "course_id": "1",
		"submission_types": []string{"online_upload"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("text", "my answer")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err == nil {
		t.Fatal("expected error for mismatched submission type, got nil")
	}
	if !strings.Contains(err.Error(), "does not accept") {
		t.Errorf("expected error about submission type mismatch, got: %s", err.Error())
	}
}

func TestSubmit_JSONMode_ReturnsEnvelopeWithSubmission(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})
	mock.On("POST", "/api/v1/courses/1/assignments/100/submissions", 200, map[string]any{
		"id": "500", "workflow_state": "submitted", "assignment_id": "100",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
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
	if sub.WorkflowState != "submitted" {
		t.Errorf("expected workflow_state 'submitted', got %q", sub.WorkflowState)
	}

	if env.Meta.Command != "assignments.submit" {
		t.Errorf("expected command 'assignments.submit', got %q", env.Meta.Command)
	}
}

func TestSubmitAuditEvent_Fields(t *testing.T) {
	event := canvas.AuditEvent{
		Time:           "2026-06-12T19:20:00Z",
		SchemaVersion:  output.SchemaVersion,
		Command:        "assignments.submit",
		Profile:        "default",
		BaseURL:        "https://school.instructure.com",
		Method:         "POST",
		Path:           "/api/v1/courses/1/assignments/100/submissions",
		Resource:       map[string]string{"course_id": "1", "assignment_id": "100"},
		RequestHash:    "sha256:abcdef",
		ResponseStatus: 200,
		DryRun:         false,
		Success:        true,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var got canvas.AuditEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if got.Command != "assignments.submit" {
		t.Errorf("command = %q, want %q", got.Command, "assignments.submit")
	}
	if got.Resource["course_id"] != "1" {
		t.Errorf("resource.course_id = %q, want %q", got.Resource["course_id"], "1")
	}
	if got.Resource["assignment_id"] != "100" {
		t.Errorf("resource.assignment_id = %q, want %q", got.Resource["assignment_id"], "100")
	}
}
