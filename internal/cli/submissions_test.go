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

func TestSubmissionsCmd_Exists(t *testing.T) {
	cmd := NewSubmissionsCmd()
	if cmd.Use != "submissions" {
		t.Errorf("expected Use 'submissions', got %q", cmd.Use)
	}
}

func TestSubmissionsCmd_HasGetSubcommand(t *testing.T) {
	cmd := NewSubmissionsCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "get" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'get' subcommand")
	}
}

func TestSubmissionsGet_ReturnsSubmission(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/10/submissions/self", 200, map[string]any{
		"id":             "500",
		"user_id":        "self",
		"assignment_id":  "10",
		"workflow_state": "submitted",
		"submitted_at":   "2026-01-15T12:00:00Z",
		"late":           false,
		"missing":        false,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSubmissionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "self")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("submissions get failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "submitted") {
		t.Errorf("expected 'submitted' in output, got: %s", output)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/assignments/10/submissions/self" {
		t.Errorf("expected request to /api/v1/courses/1/assignments/10/submissions/self, got %s", last.Path)
	}
}

func TestSubmissionsGet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/10/submissions/self", 200, map[string]any{
		"id":             "500",
		"user_id":        "self",
		"assignment_id":  "10",
		"workflow_state": "submitted",
		"submitted_at":   "2026-01-15T12:00:00Z",
		"late":           false,
		"missing":        false,
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSubmissionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "self")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("submissions get --json failed: %v", err)
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
}

func TestSubmissionsGet_SpecificUser(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/10/submissions/42", 200, map[string]any{
		"id":             "501",
		"user_id":        "42",
		"assignment_id":  "10",
		"workflow_state": "graded",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newSubmissionsGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("user", "42")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("submissions get failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "graded") {
		t.Errorf("expected 'graded' in output, got: %s", output)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/assignments/10/submissions/42" {
		t.Errorf("expected request to /api/v1/courses/1/assignments/10/submissions/42, got %s", last.Path)
	}
}

func TestSubmissionsList_ReturnsSubmissionsJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/10/submissions", 200, []map[string]any{
		{
			"id": "500", "user_id": "42", "assignment_id": "10",
			"workflow_state": "submitted",
			"user":           map[string]any{"id": "42", "name": "Alice Smith", "sortable_name": "Smith, Alice"},
		},
		{
			"id": "501", "user_id": "43", "assignment_id": "10",
			"workflow_state": "graded",
			"user":           map[string]any{"id": "43", "name": "Bob Jones", "sortable_name": "Jones, Bob"},
		},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer
	cmd := newSubmissionsListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("assignment", "10")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("list --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var subs []canvas.Submission
	if err := json.Unmarshal(dataJSON, &subs); err != nil {
		t.Fatalf("data is not []Submission: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(subs))
	}
	if subs[0].User == nil || subs[0].User.Name != "Alice Smith" {
		t.Errorf("expected user 'Alice Smith', got %v", subs[0].User)
	}
	if subs[1].User == nil || subs[1].User.Name != "Bob Jones" {
		t.Errorf("expected user 'Bob Jones', got %v", subs[1].User)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/assignments/10/submissions" {
		t.Errorf("expected path /api/v1/courses/1/assignments/10/submissions, got %s", last.Path)
	}
}

func TestSubmissionsDownload_DownloadsFiles(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/10/submissions", 200, []map[string]any{
		{
			"id": "500", "user_id": "42", "assignment_id": "10",
			"workflow_state": "submitted",
			"user":           map[string]any{"id": "42", "name": "Alice Smith", "sortable_name": "Smith, Alice"},
			"attachments": []map[string]any{
				{"id": "101", "filename": "essay.pdf", "url": mock.URL() + "/files/101/download", "size": 11},
			},
		},
		{
			"id": "501", "user_id": "43", "assignment_id": "10",
			"workflow_state": "submitted",
			"user":           map[string]any{"id": "43", "name": "Bob Jones", "sortable_name": "Jones, Bob"},
			"attachments": []map[string]any{
				{"id": "102", "filename": "report.docx", "url": mock.URL() + "/files/102/download", "size": 12},
			},
		},
	})
	mock.On("GET", "/files/101/download", 200, []byte("hello world"))
	mock.On("GET", "/files/102/download", 200, []byte("report data"))

	outDir := t.TempDir()
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)

	result, err := DownloadSubmissions(context.Background(), client, "1", "10", outDir, DownloadOptions{})
	if err != nil {
		t.Fatalf("DownloadSubmissions: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Downloaded != 2 {
		t.Errorf("Downloaded = %d, want 2", result.Downloaded)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}

	// Verify deterministic path: <assignment-id>/<sortable-name>_<user-id>/<submission-id>_<filename>
	path1 := filepath.Join(outDir, "10", "Smith, Alice_42", "500_essay.pdf")
	content, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("read %s: %v", path1, err)
	}
	if string(content) != "hello world" {
		t.Errorf("file1 content = %q, want %q", content, "hello world")
	}

	path2 := filepath.Join(outDir, "10", "Jones, Bob_43", "501_report.docx")
	content2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("read %s: %v", path2, err)
	}
	if string(content2) != "report data" {
		t.Errorf("file2 content = %q, want %q", content2, "report data")
	}

	// Verify manifest.json
	manifestPath := filepath.Join(outDir, "10", "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	var entries []ManifestEntry
	if err := json.Unmarshal(manifestData, &entries); err != nil {
		t.Fatalf("parse manifest.json: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("manifest.json has %d entries, want 2", len(entries))
	}
	if entries[0].DownloadStatus != "ok" {
		t.Errorf("entry[0].DownloadStatus = %q, want %q", entries[0].DownloadStatus, "ok")
	}

	// Verify manifest.ndjson
	ndjsonPath := filepath.Join(outDir, "10", "manifest.ndjson")
	ndjsonData, err := os.ReadFile(ndjsonPath)
	if err != nil {
		t.Fatalf("read manifest.ndjson: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(ndjsonData)), "\n")
	if len(lines) != 2 {
		t.Errorf("manifest.ndjson has %d lines, want 2", len(lines))
	}

	// Verify manifest path in result
	if result.ManifestPath != manifestPath {
		t.Errorf("ManifestPath = %q, want %q", result.ManifestPath, manifestPath)
	}
}

func TestSubmissionsDownload_NoOverwrite(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/10/submissions", 200, []map[string]any{
		{
			"id": "500", "user_id": "42", "assignment_id": "10",
			"workflow_state": "submitted",
			"user":           map[string]any{"id": "42", "name": "Alice Smith", "sortable_name": "Smith, Alice"},
			"attachments": []map[string]any{
				{"id": "101", "filename": "essay.pdf", "url": mock.URL() + "/files/101/download", "size": 11},
			},
		},
	})
	mock.On("GET", "/files/101/download", 200, []byte("new content"))

	outDir := t.TempDir()

	// Pre-create the file with different content
	dirPath := filepath.Join(outDir, "10", "Smith, Alice_42")
	os.MkdirAll(dirPath, 0755)
	existingPath := filepath.Join(dirPath, "500_essay.pdf")
	os.WriteFile(existingPath, []byte("original content"), 0644)

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)

	result, err := DownloadSubmissions(context.Background(), client, "1", "10", outDir, DownloadOptions{NoOverwrite: true})
	if err != nil {
		t.Fatalf("DownloadSubmissions: %v", err)
	}

	// File should not be overwritten
	content, _ := os.ReadFile(existingPath)
	if string(content) != "original content" {
		t.Errorf("file was overwritten: got %q, want %q", content, "original content")
	}

	// Result should count skipped as downloaded
	if result.Downloaded != 1 {
		t.Errorf("Downloaded = %d, want 1 (skipped)", result.Downloaded)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}

	// Manifest should show skipped status
	manifestPath := filepath.Join(outDir, "10", "manifest.json")
	manifestData, _ := os.ReadFile(manifestPath)
	var entries []ManifestEntry
	json.Unmarshal(manifestData, &entries)
	if len(entries) != 1 {
		t.Fatalf("manifest has %d entries, want 1", len(entries))
	}
	if entries[0].DownloadStatus != "skipped" {
		t.Errorf("entry status = %q, want %q", entries[0].DownloadStatus, "skipped")
	}
}

func TestSubmissionsDownload_PartialFailure(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/assignments/10/submissions", 200, []map[string]any{
		{
			"id": "500", "user_id": "42", "assignment_id": "10",
			"workflow_state": "submitted",
			"user":           map[string]any{"id": "42", "name": "Alice Smith", "sortable_name": "Smith, Alice"},
			"attachments": []map[string]any{
				{"id": "101", "filename": "essay.pdf", "url": mock.URL() + "/files/101/download", "size": 11},
			},
		},
		{
			"id": "501", "user_id": "43", "assignment_id": "10",
			"workflow_state": "submitted",
			"user":           map[string]any{"id": "43", "name": "Bob Jones", "sortable_name": "Jones, Bob"},
			"attachments": []map[string]any{
				{"id": "102", "filename": "report.docx", "url": mock.URL() + "/files/102/download", "size": 12},
			},
		},
	})
	mock.On("GET", "/files/101/download", 200, []byte("hello world"))
	mock.On("GET", "/files/102/download", 500, map[string]any{"error": "server error"})

	outDir := t.TempDir()
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)

	result, err := DownloadSubmissions(context.Background(), client, "1", "10", outDir, DownloadOptions{})
	if err == nil {
		t.Fatal("expected error for partial failure, got nil")
	}

	pfErr, ok := err.(*PartialFailureError)
	if !ok {
		t.Fatalf("expected PartialFailureError, got %T: %v", err, err)
	}
	if pfErr.ExitCode() != 8 {
		t.Errorf("ExitCode() = %d, want 8", pfErr.ExitCode())
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Downloaded != 1 {
		t.Errorf("Downloaded = %d, want 1", result.Downloaded)
	}
	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}

	// Verify successful file was downloaded
	path1 := filepath.Join(outDir, "10", "Smith, Alice_42", "500_essay.pdf")
	content, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("read successful download: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("successful file content = %q, want %q", content, "hello world")
	}

	// Manifest should have both entries with different statuses
	manifestPath := filepath.Join(outDir, "10", "manifest.json")
	manifestData, _ := os.ReadFile(manifestPath)
	var entries []ManifestEntry
	json.Unmarshal(manifestData, &entries)
	if len(entries) != 2 {
		t.Fatalf("manifest has %d entries, want 2", len(entries))
	}

	var okCount, errCount int
	for _, e := range entries {
		switch e.DownloadStatus {
		case "ok":
			okCount++
		case "error":
			errCount++
		}
	}
	if okCount != 1 {
		t.Errorf("manifest ok entries = %d, want 1", okCount)
	}
	if errCount != 1 {
		t.Errorf("manifest error entries = %d, want 1", errCount)
	}
}
