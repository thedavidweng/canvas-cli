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

// secretToken is a distinctive token used across all redaction tests.
// It must never appear in any output stream.
const secretToken = "canvas-cli-redact-test-s3cr3t-t0k3n-xyz"

// redactCfg returns a ResolvedConfig wired to the mock server with the secret token.
func redactCfg(mock *testutil.MockCanvas) *config.ResolvedConfig {
	return &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   secretToken,
		Profile: "default",
	}
}

// assertNotContains fails the test if haystack contains the secret token.
func assertNotContains(t *testing.T, label, haystack string) {
	t.Helper()
	if strings.Contains(haystack, secretToken) {
		t.Errorf("%s: token value leaked into output", label)
	}
}

// ---------------------------------------------------------------------------
// Token never in stdout
// ---------------------------------------------------------------------------

func TestRedaction_TokenNeverInStdout_MeGet(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	var stdout bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "me get stdout", stdout.String())
}

func TestRedaction_TokenNeverInStdout_MeGetJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	var stdout bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("json", "true")

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "me get --json stdout", stdout.String())
}

func TestRedaction_TokenNeverInStdout_CoursesList(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Test Course", "course_code": "TC101", "workflow_state": "available"},
	})

	var stdout bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "courses list stdout", stdout.String())
}

func TestRedaction_TokenNeverInStdout_CoursesListJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Test Course", "course_code": "TC101", "workflow_state": "available"},
	})

	var stdout bytes.Buffer
	cmd := newCoursesListCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("json", "true")

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "courses list --json stdout", stdout.String())
}

func TestRedaction_TokenNeverInStdout_AssignmentsList(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Essay 1", "course_id": "1", "published": true, "points_possible": 100},
	})

	var stdout bytes.Buffer
	cmd := newAssignmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "assignments list stdout", stdout.String())
}

func TestRedaction_TokenNeverInStdout_ApiGet(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	var stdout bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)

	_ = cmd.RunE(cmd, []string{"/api/v1/courses"})
	assertNotContains(t, "api get stdout", stdout.String())
}

func TestRedaction_TokenNeverInStdout_ApiGetJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	var stdout bytes.Buffer
	cmd := newApiGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("json", "true")

	_ = cmd.RunE(cmd, []string{"/api/v1/courses"})
	assertNotContains(t, "api get --json stdout", stdout.String())
}

// ---------------------------------------------------------------------------
// Token never in stderr
// ---------------------------------------------------------------------------

func TestRedaction_TokenNeverInStderr_OnAuthError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	var stdout, stderr bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "stderr on auth error", stderr.String())
	assertNotContains(t, "stdout on auth error", stdout.String())
}

func TestRedaction_TokenNeverInStderr_OnNetworkError(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost:0",
		Token:   secretToken,
		Profile: "default",
	}

	var stdout, stderr bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "stderr on network error", stderr.String())
	assertNotContains(t, "stdout on network error", stdout.String())
}

func TestRedaction_TokenNeverInStderr_OnPermissionDenied(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 403, map[string]any{
		"errors": []map[string]any{{"message": "Forbidden"}},
	})

	var stdout, stderr bytes.Buffer
	cmd := newAssignmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "stderr on permission denied", stderr.String())
	assertNotContains(t, "stdout on permission denied", stdout.String())
}

func TestRedaction_TokenNeverInStderr_OnRateLimit(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.SetRetryAfter(30)

	var stdout, stderr bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "stderr on rate limit", stderr.String())
	assertNotContains(t, "stdout on rate limit", stdout.String())
}

// ---------------------------------------------------------------------------
// Token never in audit log
// ---------------------------------------------------------------------------

func TestRedaction_TokenNeverInAuditLog(t *testing.T) {
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
		Token:        secretToken,
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

	auditContent := string(data)
	assertNotContains(t, "audit log", auditContent)

	// Also verify the audit event structure is correct (no token in any field).
	lines := strings.Split(strings.TrimSpace(auditContent), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		var event canvas.AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("audit line %d is not valid JSON: %v", i, err)
		}
		// Token should never appear in any event field.
		assertNotContains(t, "audit event JSON", line)
	}
}

func TestRedaction_TokenNeverInAuditLog_Submit(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})
	mock.On("POST", "/api/v1/courses/1/assignments/100/submissions", 200, map[string]any{
		"id": "500", "workflow_state": "submitted", "assignment_id": "100",
	})

	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        secretToken,
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    auditPath,
	}

	var stdout bytes.Buffer
	cmd := newAssignmentsSubmitCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100", "my answer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	assertNotContains(t, "audit log (submit)", string(data))
}

// ---------------------------------------------------------------------------
// Token redacted in error messages
// ---------------------------------------------------------------------------

func TestRedaction_TokenRedactedInErrorMessages_401(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	// Non-JSON mode: error is returned as Go error.
	var stdout bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		assertNotContains(t, "Go error on 401", err.Error())
	}

	// JSON mode: error is written to stdout.
	stdout.Reset()
	cmd2 := newMeGetCmd()
	cmd2.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd2.SetOut(&stdout)
	_ = cmd2.Flags().Set("json", "true")

	_ = cmd2.RunE(cmd2, nil)
	assertNotContains(t, "JSON error on 401", stdout.String())
}

func TestRedaction_TokenRedactedInErrorMessages_403(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 403, map[string]any{
		"errors": []map[string]any{{"message": "Forbidden"}},
	})

	var stdout bytes.Buffer
	cmd := newAssignmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	_ = cmd.RunE(cmd, nil)
	assertNotContains(t, "JSON error on 403", stdout.String())
}

func TestRedaction_TokenRedactedInErrorMessages_NetworkError(t *testing.T) {
	cfg := &config.ResolvedConfig{
		BaseURL: "http://localhost:0",
		Token:   secretToken,
		Profile: "default",
	}

	// Non-JSON mode.
	var stdout bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&stdout)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		assertNotContains(t, "Go error on network error", err.Error())
	}

	// JSON mode.
	stdout.Reset()
	cmd2 := newMeGetCmd()
	cmd2.SetContext(WithConfig(context.Background(), cfg))
	cmd2.SetOut(&stdout)
	_ = cmd2.Flags().Set("json", "true")

	_ = cmd2.RunE(cmd2, nil)
	assertNotContains(t, "JSON error on network error", stdout.String())
}

func TestRedaction_TokenRedactedInErrorMessages_SafetyBlocked(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1",
		"submission_types": []string{"online_text_entry"},
	})

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    secretToken,
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
	if err != nil {
		assertNotContains(t, "Go error on safety blocked", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Token never in JSON error envelope body
// ---------------------------------------------------------------------------

func TestRedaction_TokenNeverInJSONErrorEnvelope(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/users/self", 401, map[string]any{
		"errors": []map[string]any{{"message": "Unauthorized"}},
	})

	var stdout bytes.Buffer
	cmd := newMeGetCmd()
	cmd.SetContext(WithConfig(context.Background(), redactCfg(mock)))
	cmd.SetOut(&stdout)
	_ = cmd.Flags().Set("json", "true")

	_ = cmd.RunE(cmd, nil)

	raw := stdout.String()
	assertNotContains(t, "JSON error envelope", raw)

	// Parse and verify the error envelope doesn't contain the token in any field.
	var env canvas.Envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if env.Error != nil {
		assertNotContains(t, "error.code", env.Error.Code)
		assertNotContains(t, "error.message", env.Error.Message)
		assertNotContains(t, "error.category", env.Error.Category)
	}
	// Re-marshal and check again (catches any field we might have missed).
	fullJSON, _ := json.Marshal(env)
	assertNotContains(t, "full envelope re-marshal", string(fullJSON))
}

// ---------------------------------------------------------------------------
// Comprehensive: token never leaks in any stream for any command variant
// ---------------------------------------------------------------------------

func TestRedaction_Comprehensive_AllCommands(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Register routes for commands that hit different endpoints.
	mock.On("GET", "/api/v1/courses", 200, []map[string]any{
		{"id": "1", "name": "Test Course", "course_code": "TC101", "workflow_state": "available"},
	})
	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Essay 1", "course_id": "1", "published": true, "points_possible": 100},
	})

	cfg := redactCfg(mock)

	tests := []struct {
		name    string
		factory func() *cobra.Command
		args    []string
		flags   map[string]string
	}{
		{"me get", newMeGetCmd, nil, nil},
		{"me get --json", newMeGetCmd, nil, map[string]string{"json": "true"}},
		{"courses list", newCoursesListCmd, nil, nil},
		{"courses list --json", newCoursesListCmd, nil, map[string]string{"json": "true"}},
		{"assignments list", newAssignmentsListCmd, nil, map[string]string{"course": "1"}},
		{"assignments list --json", newAssignmentsListCmd, nil, map[string]string{"course": "1", "json": "true"}},
		{"api get", newApiGetCmd, []string{"/api/v1/courses"}, nil},
		{"api get --json", newApiGetCmd, []string{"/api/v1/courses"}, map[string]string{"json": "true"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := tc.factory()
			cmd.SetContext(WithConfig(context.Background(), cfg))
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tc.args)
			for k, v := range tc.flags {
				_ = cmd.Flags().Set(k, v)
			}

			_ = cmd.RunE(cmd, tc.args)

			assertNotContains(t, tc.name+" stdout", stdout.String())
			assertNotContains(t, tc.name+" stderr", stderr.String())
		})
	}
}
