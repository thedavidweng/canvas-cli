package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// exitCode extracts the exit code from an error, returning 0 if err is nil
// or does not implement ExitCode() int.
func exitCode(err error) int {
	if err == nil {
		return output.CodeSuccess
	}
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	// Check for *safety.SafetyError which has ExitCode as a field.
	var se *safety.SafetyError
	if errors.As(err, &se) {
		return se.ExitCode
	}
	return output.CodeGenericError
}

// ---------------------------------------------------------------------------
// 1. JSON Output Shape Tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 2. Exit Code Tests
// ---------------------------------------------------------------------------

func TestGolden_ExitCode_Success(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if code := exitCode(err); code != output.CodeSuccess {
		t.Errorf("expected exit code %d, got %d (err=%v)", output.CodeSuccess, code, err)
	}
}

func TestGolden_ExitCode_PermissionDenied(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 403, map[string]any{
		"errors": []map[string]any{{"message": "Forbidden"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	// Use api get which properly maps HTTP status codes to error codes.
	var buf bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses/1/assignments"})
	if err != nil {
		t.Fatalf("in JSON mode command should write error to stdout, not return error: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if env.OK {
		t.Error("expected ok=false for permission denied")
	}
	if env.Error == nil {
		t.Fatal("expected error field in envelope")
	}
	if env.Error.Code != "CANVAS_PERMISSION_DENIED" {
		t.Errorf("expected error code %q, got %q", "CANVAS_PERMISSION_DENIED", env.Error.Code)
	}
	if env.Error.Category != "permission" {
		t.Errorf("expected category %q, got %q", "permission", env.Error.Category)
	}
}

func TestGolden_ExitCode_RateLimitExhausted(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.SetRetryAfter(30)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	// In JSON mode, the 429 is written to the envelope on stdout (command returns nil).
	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("JSON mode should not return Go error for rate limit, got: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected JSON envelope on stdout for rate limit error")
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if env.OK {
		t.Error("expected ok=false for rate limit error")
	}
	if env.Error == nil {
		t.Fatal("expected error field in envelope")
	}

	// In non-JSON mode, rate limit errors are returned as Go errors.
	var buf2 bytes.Buffer
	cmd2 := newMeGetCmd()
	cmd2.SetContext(WithConfig(context.Background(), cfg))
	cmd2.SetOut(&buf2)

	err2 := cmd2.RunE(cmd2, nil)
	if err2 == nil {
		t.Fatal("expected Go error for 429 in non-JSON mode")
	}
}

func TestGolden_ExitCode_SafetyBlocked(t *testing.T) {
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
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err == nil {
		t.Fatal("expected error in read-only mode")
	}

	code := exitCode(err)
	if code != output.CodeSafetyBlocked {
		t.Errorf("expected exit code %d (safety blocked), got %d (err=%v)", output.CodeSafetyBlocked, code, err)
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("expected 'read-only' in error message, got: %s", err.Error())
	}
}

func TestGolden_ExitCode_PartialFailure(t *testing.T) {
	// Verify the constant is correctly defined.
	if output.CodePartialFailure != 8 {
		t.Errorf("expected CodePartialFailure=8, got %d", output.CodePartialFailure)
	}
}

// ---------------------------------------------------------------------------
// 3. Safety Gate End-to-End Tests
// ---------------------------------------------------------------------------

func TestGolden_ReadOnly_BlocksWrite(t *testing.T) {
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

	var stdout bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err == nil {
		t.Fatal("expected --read-only to block the write operation")
	}

	exitErr, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("expected error with ExitCode(), got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Errorf("expected exit code 7, got %d", exitErr.ExitCode())
	}

	// Verify no POST request was sent.
	for _, req := range mock.RequestLog() {
		if req.Method == "POST" {
			t.Errorf("--read-only should block all writes, but POST was sent to %s", req.Path)
		}
	}
}

func TestGolden_DryRun_ShowsPreviewNoRequest(t *testing.T) {
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

	var stdout bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err != nil {
		t.Fatalf("expected no error for --dry-run, got: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("expected 'DRY RUN' in preview output, got: %s", out)
	}
	if !strings.Contains(out, "No mutation sent") {
		t.Errorf("expected 'No mutation sent' in preview output, got: %s", out)
	}
	if !strings.Contains(out, "POST") {
		t.Errorf("expected HTTP method 'POST' in preview, got: %s", out)
	}

	// Verify no mutation (POST) was sent.
	for _, req := range mock.RequestLog() {
		if req.Method == "POST" {
			t.Errorf("--dry-run should not send mutations, but POST was sent to %s", req.Path)
		}
	}
}

func TestGolden_Confirm_ExecutesMutation(t *testing.T) {
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

	var stdout bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err != nil {
		t.Fatalf("expected no error with --confirm, got: %v", err)
	}

	// Verify POST was sent.
	found := false
	for _, req := range mock.RequestLog() {
		if req.Method == "POST" && strings.HasSuffix(req.Path, "/submissions") {
			found = true
			break
		}
	}
	if !found {
		t.Error("--confirm should execute the mutation, but no POST to /submissions was found")
	}

	// Verify success output.
	out := stdout.String()
	if !strings.Contains(out, "submitted") && !strings.Contains(out, "500") {
		t.Errorf("expected success message in output, got: %s", out)
	}
}

func TestGolden_ReadOnly_BlocksUpdateEvenWithConfirm(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var stdout bytes.Buffer
	cmd := newAssignmentsUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("due-at", "2026-07-01T23:59:00Z")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err == nil {
		t.Fatal("expected --read-only to block update even with --confirm")
	}

	code := exitCode(err)
	if code != output.CodeSafetyBlocked {
		t.Errorf("expected exit code %d, got %d", output.CodeSafetyBlocked, code)
	}
}

// ---------------------------------------------------------------------------
// 4. Token Redaction Tests (golden-level)
// ---------------------------------------------------------------------------

func TestGolden_TokenNotInStdout(t *testing.T) {
	secret := "super-secret-token-abc123"

	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   secret,
		Profile: "default",
	}

	commands := []struct {
		name    string
		factory func() *cobra.Command
		args    []string
		flags   map[string]string
	}{
		{"me get", newMeGetCmd, nil, map[string]string{"json": "true"}},
		{"courses list", newCoursesListCmd, nil, map[string]string{"json": "true"}},
		{"api get", newApiGetCmd, []string{"/api/v1/courses"}, map[string]string{"json": "true"}},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			cmd := tc.factory()
			cmd.SetContext(WithConfig(context.Background(), cfg))
			cmd.SetOut(&stdout)
			cmd.SetArgs(tc.args)
			for k, v := range tc.flags {
				_ = cmd.Flags().Set(k, v)
			}

			_ = cmd.RunE(cmd, tc.args)

			if strings.Contains(stdout.String(), secret) {
				t.Errorf("token leaked to stdout for %s", tc.name)
			}
		})
	}
}

func TestGolden_TokenNotInStderr(t *testing.T) {
	secret := "super-secret-token-abc123"

	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   secret,
		Profile: "default",
	}

	var stdout, stderr bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	_ = cmd.RunE(cmd, nil)

	if strings.Contains(stderr.String(), secret) {
		t.Errorf("token leaked to stderr")
	}
}

func TestGolden_TokenNotInAuditLog(t *testing.T) {
	secret := "super-secret-token-abc123"

	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"due_at": "2026-07-01T23:59:00Z", "published": true,
	})
	mock.On("PUT", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"due_at": "2026-07-01T23:59:00Z", "published": true,
	})

	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        secret,
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	var stdout bytes.Buffer
	cmd := newAssignmentsUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("due-at", "2026-07-01T23:59:00Z")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	if strings.Contains(string(data), secret) {
		t.Errorf("token leaked to audit log")
	}
}

func TestGolden_TokenNotInErrorMessages(t *testing.T) {
	secret := "super-secret-token-abc123"

	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Force a 401 error.
	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   secret,
		Profile: "default",
	}

	// Test non-JSON mode (Go error returned).
	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("token leaked in Go error message: %s", err.Error())
		}
	}

	// Test JSON mode (error written to stdout).
	buf.Reset()
	cmd2 := newMeGetCmd()
	cmd2.SetContext(WithConfig(context.Background(), cfg))
	cmd2.SetOut(&buf)
	_ = cmd2.Flags().Set("json", "true")

	_ = cmd2.RunE(cmd2, nil)
	if strings.Contains(buf.String(), secret) {
		t.Errorf("token leaked in JSON error output")
	}
}

// ---------------------------------------------------------------------------
// 5. Stdout/Stderr Separation Tests
// ---------------------------------------------------------------------------

func TestGolden_StdoutStderr_DataGoesToStdout(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var stdout, stderr bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Data should be on stdout.
	if stdout.Len() == 0 {
		t.Error("expected data on stdout, got empty")
	}
	if !strings.Contains(stdout.String(), "Test User") {
		t.Errorf("expected user data on stdout, got: %s", stdout.String())
	}
}

func TestGolden_StdoutStderr_JSONMode_OnlyJSONOnStdout(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var stdout, stderr bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stdout should contain only valid JSON.
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		t.Fatal("expected JSON on stdout, got empty")
	}

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nraw: %s", err, out)
	}

	if !strings.HasPrefix(out, "{") {
		t.Errorf("expected JSON to start with '{', got: %s", out[:min(20, len(out))])
	}
}

func TestGolden_StdoutStderr_JSONMode_StderrIsClean(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var stdout, stderr bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stderr should be empty for successful commands.
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr for success, got: %s", stderr.String())
	}
}

func TestGolden_StdoutStderr_CoursesList_JSON_PureJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Intro to CS", "course_code": "CS101", "workflow_state": "available"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var stdout, stderr bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify stdout is valid JSON.
	var env canvas.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}

	// Stderr should be empty.
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestGolden_StdoutStderr_ApiGet_JSON_PureJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var stdout, stderr bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"/api/v1/courses"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify stdout is valid JSON.
	var env canvas.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}

	// Stderr should be empty.
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestGolden_ExitCode_AuthError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "bad-token",
		Profile: "default",
	}

	// In JSON mode, auth errors are written to the envelope on stdout (command returns nil).
	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("JSON mode should not return Go error for API errors, got: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if env.OK {
		t.Error("expected ok=false for auth error")
	}
	if env.Error == nil {
		t.Fatal("expected error field in envelope")
	}
	if env.Error.Code != "CANVAS_AUTH_ERROR" {
		t.Errorf("expected error code %q, got %q", "CANVAS_AUTH_ERROR", env.Error.Code)
	}
	if env.Error.Category != "auth" {
		t.Errorf("expected category %q, got %q", "auth", env.Error.Category)
	}
	if env.Meta.RequestID == "" {
		t.Error("expected request_id in error envelope meta")
	}

	// In non-JSON mode, auth errors are returned as Go errors.
	var buf2 bytes.Buffer
	cmd2 := newMeGetCmd()
	cmd2.SetContext(WithConfig(context.Background(), cfg))
	cmd2.SetOut(&buf2)

	err2 := cmd2.RunE(cmd2, nil)
	if err2 == nil {
		t.Fatal("expected Go error for 401 in non-JSON mode")
	}
}

func TestGolden_ExitCode_NetworkError(t *testing.T) {
	// Use a URL that will fail to connect.
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost:0",
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	// Network errors in JSON mode are written to stdout as an envelope.
	if err == nil {
		// If no error returned, it should be in stdout as JSON error envelope.
		if buf.Len() > 0 {
			var env canvas.Envelope
			if jsonErr := json.Unmarshal(buf.Bytes(), &env); jsonErr == nil {
				if env.Error != nil && env.Error.Code == "CANVAS_NETWORK_ERROR" {
					return // Expected behavior
				}
			}
		}
		t.Fatal("expected network error")
	}
}
