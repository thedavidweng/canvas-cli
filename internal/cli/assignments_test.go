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

func TestAssignmentsList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Essay 1", "course_id": "1", "published": true, "points_possible": 100},
		{"id": "101", "name": "Quiz 1", "course_id": "1", "published": true, "points_possible": 50},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("assignments list --course 1 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var assignments []canvas.Assignment
	if err := json.Unmarshal(dataJSON, &assignments); err != nil {
		t.Fatalf("data is not []Assignment: %v", err)
	}
	if len(assignments) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(assignments))
	}
	if assignments[0].Name != "Essay 1" {
		t.Errorf("expected assignment name 'Essay 1', got %q", assignments[0].Name)
	}
}

func TestAssignmentsList_BucketFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Past Assignment", "course_id": "1", "published": true},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("bucket", "past")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("assignments list --bucket past failed: %v", err)
	}

	// Verify the bucket query param was sent
	last := mock.LastRequest()
	if last.Query.Get("bucket") != "past" {
		t.Errorf("expected query param bucket=past, got: %v", last.Query)
	}
}

func TestAssignmentsList_SortFlag(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Essay 1", "course_id": "1", "published": true},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("sort", "due_at")
	_ = cmd.Flags().Set("order", "desc")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("assignments list --sort due_at failed: %v", err)
	}

	// Verify query params were sent
	last := mock.LastRequest()
	if last.Query.Get("sort") != "due_at" {
		t.Errorf("expected query param sort=due_at, got: %v", last.Query)
	}
	if last.Query.Get("order") != "desc" {
		t.Errorf("expected query param order=desc, got: %v", last.Query)
	}
}

func TestAssignmentsGet_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id": "100", "name": "Essay 1", "course_id": "1", "published": true, "points_possible": 100,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("assignments get --course 1 100 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var assignment canvas.Assignment
	if err := json.Unmarshal(dataJSON, &assignment); err != nil {
		t.Fatalf("data is not Assignment: %v", err)
	}
	if assignment.ID != "100" {
		t.Errorf("expected assignment ID '100', got %q", assignment.ID)
	}
	if assignment.Name != "Essay 1" {
		t.Errorf("expected assignment name 'Essay 1', got %q", assignment.Name)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/assignments/100" {
		t.Errorf("expected request to /api/v1/courses/1/assignments/100, got %s", last.Path)
	}
}

func TestAssignmentGroupsList_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignment_groups", 200, []map[string]any{
		{"id": "10", "name": "Homework", "position": 1, "group_weight": 40},
		{"id": "11", "name": "Exams", "position": 2, "group_weight": 60},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentGroupsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("assignment-groups list --course 1 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var groups []canvas.AssignmentGroup
	if err := json.Unmarshal(dataJSON, &groups); err != nil {
		t.Fatalf("data is not []AssignmentGroup: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "Homework" {
		t.Errorf("expected group name 'Homework', got %q", groups[0].Name)
	}

	// Verify the correct API path was hit
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/assignment_groups" {
		t.Errorf("expected request to /api/v1/courses/1/assignment_groups, got %s", last.Path)
	}
}

func TestAssignmentsList_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments", 200, []map[string]any{
		{"id": "100", "name": "Essay 1", "course_id": "1", "published": true, "points_possible": 100},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("assignments list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Essay 1") {
		t.Errorf("expected assignment name in human output, got: %s", output)
	}
}

func TestAssignmentsUpdate_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newAssignmentsUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("due-at", "2026-07-01T23:59:00Z")
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("assignments update --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PUT") {
		t.Errorf("expected 'PUT' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/assignments/100") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "2026-07-01T23:59:00Z") {
		t.Errorf("expected due-at time in dry-run output, got: %s", output)
	}
	// Verify no actual request was made
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestAssignmentsUpdate_ConfirmSendsPUT(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id":        "100",
		"name":      "Essay 1",
		"course_id": "1",
		"due_at":    "2026-07-01T23:59:00Z",
		"published": true,
	})

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newAssignmentsUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("due-at", "2026-07-01T23:59:00Z")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("assignments update --confirm failed: %v", err)
	}

	last := mock.LastRequest()
	if last == nil {
		t.Fatal("expected at least one request")
	}
	if last.Method != "PUT" {
		t.Errorf("expected PUT method, got %s", last.Method)
	}
	if last.Path != "/api/v1/courses/1/assignments/100" {
		t.Errorf("expected path /api/v1/courses/1/assignments/100, got %s", last.Path)
	}
	if !strings.Contains(last.Body, "2026-07-01T23:59:00Z") {
		t.Errorf("expected due_at in request body, got: %s", last.Body)
	}

	output := buf.String()
	if !strings.Contains(output, "updated") {
		t.Errorf("expected success message in output, got: %s", output)
	}
}

func TestAssignmentsUpdate_ReadOnlyReturnsExit7(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL:  mock.URL(),
		Token:    "test-token",
		Profile:  "default",
		ReadOnly: true,
	}

	var buf bytes.Buffer
	cmd := newAssignmentsUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("due-at", "2026-07-01T23:59:00Z")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
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

func TestAssignmentsUpdate_WritesAuditLog(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("PUT", "/api/v1/courses/1/assignments/100", 200, map[string]any{
		"id":        "100",
		"name":      "Essay 1",
		"course_id": "1",
		"due_at":    "2026-07-01T23:59:00Z",
		"published": true,
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
	cmd := newAssignmentsUpdateCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("due-at", "2026-07-01T23:59:00Z")
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, []string{"100"})
	if err != nil {
		t.Fatalf("assignments update --confirm failed: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("audit log is empty")
	}
	if !strings.Contains(string(data), "assignments.update") {
		t.Errorf("expected 'assignments.update' in audit log, got: %s", string(data))
	}
}
