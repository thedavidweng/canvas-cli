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

func TestFilesCmd_Exists(t *testing.T) {
	cmd := NewFilesCmd()
	if cmd.Use != "files" {
		t.Errorf("expected Use 'files', got %q", cmd.Use)
	}
}

func TestFilesCmd_HasListSubcommand(t *testing.T) {
	cmd := NewFilesCmd()
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

func TestFilesCmd_HasDownloadSubcommand(t *testing.T) {
	cmd := NewFilesCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Name() == "download" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'download' subcommand")
	}
}

func TestFilesList_ReturnsFiles(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/files", 200, []map[string]any{
		{
			"id":           "10",
			"display_name": "syllabus.pdf",
			"filename":     "syllabus.pdf",
			"size":         1024,
			"content_type": "application/pdf",
		},
		{
			"id":           "11",
			"display_name": "slides.pptx",
			"filename":     "slides.pptx",
			"size":         2048,
			"content_type": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "syllabus.pdf") {
		t.Errorf("expected syllabus.pdf in output, got: %s", output)
	}
	if !strings.Contains(output, "slides.pptx") {
		t.Errorf("expected slides.pptx in output, got: %s", output)
	}

	// Verify the request hit the right endpoint
	last := mock.LastRequest()
	if last.Path != "/api/v1/courses/1/files" {
		t.Errorf("expected request to /api/v1/courses/1/files, got %s", last.Path)
	}
}

func TestFilesList_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/files", 200, []map[string]any{
		{
			"id":           "10",
			"display_name": "syllabus.pdf",
			"filename":     "syllabus.pdf",
			"size":         1024,
			"content_type": "application/pdf",
		},
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesListCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files list failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

func TestFilesDownload_DownloadsFile(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Register file metadata endpoint
	mock.On("GET", "/api/v1/files/10", 200, map[string]any{
		"id":           "10",
		"display_name": "syllabus.pdf",
		"filename":     "syllabus.pdf",
		"url":          mock.URL() + "/files/10/download",
		"size":         1024,
		"content_type": "application/pdf",
	})

	// Register download endpoint
	mock.On("GET", "/files/10/download", 200, "file-content-here")

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "syllabus.pdf")

	var buf bytes.Buffer
	cmd := newFilesDownloadCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("out", outPath)

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("files download failed: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "file-content-here" {
		t.Errorf("expected file content 'file-content-here', got %q", string(data))
	}

	output := buf.String()
	if !strings.Contains(output, "Downloaded") {
		t.Errorf("expected 'Downloaded' in output, got: %s", output)
	}
}

func TestFilesDownload_RespectsNoOverwrite(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "existing.pdf")

	// Create an existing file
	if err := os.WriteFile(outPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := newFilesDownloadCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("out", outPath)
	_ = cmd.Flags().Set("no-overwrite", "true")

	err := cmd.RunE(cmd, []string{"10"})
	if err == nil {
		t.Fatal("expected error when file exists and --no-overwrite is set")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

// --- files get ---

func TestFilesGet_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/files/10", 200, map[string]any{
		"id":           "10",
		"display_name": "syllabus.pdf",
		"filename":     "syllabus.pdf",
		"size":         1024,
		"content_type": "application/pdf",
		"created_at":   "2026-01-01T00:00:00Z",
		"updated_at":   "2026-01-01T00:00:00Z",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("files get 10 --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}

	dataJSON, _ := json.Marshal(env.Data)
	var file canvas.File
	if err := json.Unmarshal(dataJSON, &file); err != nil {
		t.Fatalf("data is not File: %v", err)
	}
	if file.ID != "10" {
		t.Errorf("expected file ID '10', got %q", file.ID)
	}
	if file.DisplayName != "syllabus.pdf" {
		t.Errorf("expected display name 'syllabus.pdf', got %q", file.DisplayName)
	}

	last := mock.LastRequest()
	if last.Path != "/api/v1/files/10" {
		t.Errorf("expected request to /api/v1/files/10, got %s", last.Path)
	}
}

func TestFilesGet_HumanMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/files/10", 200, map[string]any{
		"id":           "10",
		"display_name": "syllabus.pdf",
		"filename":     "syllabus.pdf",
		"size":         1024,
		"content_type": "application/pdf",
		"created_at":   "2026-01-01T00:00:00Z",
		"updated_at":   "2026-01-01T00:00:00Z",
	})

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesGetCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"10"})
	if err != nil {
		t.Fatalf("files get 10 failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "syllabus.pdf") {
		t.Errorf("expected 'syllabus.pdf' in output, got: %s", output)
	}
	if !strings.Contains(output, "1024") {
		t.Errorf("expected size '1024' in output, got: %s", output)
	}
	if !strings.Contains(output, "application/pdf") {
		t.Errorf("expected content type in output, got: %s", output)
	}
}

// --- files upload ---

func TestFilesUpload_DryRunShowsPreview(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesUploadCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("file", filePath)
	_ = cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files upload --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected 'POST' in dry-run output, got: %s", output)
	}
	if !strings.Contains(output, "/api/v1/courses/1/files") {
		t.Errorf("expected endpoint path in dry-run output, got: %s", output)
	}
	if mock.RequestCount() != 0 {
		t.Errorf("dry-run should not make HTTP requests, got %d", mock.RequestCount())
	}
}

func TestFilesUpload_ConfirmUploads(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Step 1: init returns upload_url and params
	mock.OnUploadInit("POST", "/api/v1/courses/1/files", "/uploads/123", map[string]string{
		"token": "abc",
	})
	// Step 2: receive the upload
	mock.On("POST", "/uploads/123", 201, map[string]any{
		"id":           "555",
		"display_name": "test.txt",
		"filename":     "test.txt",
	})

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("hello world"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newFilesUploadCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("file", filePath)
	_ = cmd.Flags().Set("confirm", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files upload --confirm failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Uploaded") {
		t.Errorf("expected 'Uploaded' in output, got: %s", output)
	}
	if !strings.Contains(output, "555") {
		t.Errorf("expected file ID '555' in output, got: %s", output)
	}
}

func TestFilesUpload_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.OnUploadInit("POST", "/api/v1/courses/1/files", "/uploads/123", map[string]string{
		"token": "abc",
	})
	mock.On("POST", "/uploads/123", 201, map[string]any{
		"id":           "555",
		"display_name": "test.txt",
		"filename":     "test.txt",
	})

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("hello world"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL:      mock.URL(),
		Token:        "test-token",
		Profile:      "default",
		AuditEnabled: true,
		AuditPath:    filepath.Join(t.TempDir(), "audit.jsonl"),
	}

	var buf bytes.Buffer
	cmd := newFilesUploadCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("file", filePath)
	_ = cmd.Flags().Set("confirm", "true")
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files upload --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

// --- files download-course ---

func TestFilesDownloadCourse_DownloadsFiles(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/files", 200, []map[string]any{
		{
			"id":           "10",
			"display_name": "syllabus.pdf",
			"filename":     "syllabus.pdf",
			"size":         1024,
			"content_type": "application/pdf",
		},
		{
			"id":           "11",
			"display_name": "slides.pptx",
			"filename":     "slides.pptx",
			"size":         2048,
			"content_type": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
	})

	// DownloadFile first fetches metadata to get the download URL.
	mock.On("GET", "/api/v1/files/10", 200, map[string]any{
		"id":       "10",
		"filename": "syllabus.pdf",
		"url":      mock.URL() + "/files/10/download",
	})
	mock.On("GET", "/api/v1/files/11", 200, map[string]any{
		"id":       "11",
		"filename": "slides.pptx",
		"url":      mock.URL() + "/files/11/download",
	})

	mock.On("GET", "/files/10/download", 200, []byte("file content 1"))
	mock.On("GET", "/files/11/download", 200, []byte("file content 2"))

	outDir := t.TempDir()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesDownloadCourseCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("out", outDir)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files download-course failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Downloaded 2/2") {
		t.Errorf("expected 'Downloaded 2/2' in output, got: %s", output)
	}

	// Verify manifest was created
	manifestPath := filepath.Join(outDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("expected manifest.json to be created")
	}
}

func TestFilesDownloadCourse_JSONMode(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/files", 200, []map[string]any{
		{
			"id":           "10",
			"display_name": "syllabus.pdf",
			"filename":     "syllabus.pdf",
			"size":         1024,
			"content_type": "application/pdf",
		},
	})

	mock.On("GET", "/api/v1/files/10", 200, map[string]any{
		"id":       "10",
		"filename": "syllabus.pdf",
		"url":      mock.URL() + "/files/10/download",
	})
	mock.On("GET", "/files/10/download", 200, []byte("file content"))

	outDir := t.TempDir()

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesDownloadCourseCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("out", outDir)
	_ = cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files download-course --json failed: %v", err)
	}

	var env canvas.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if !env.OK {
		t.Error("expected ok:true")
	}
}

func TestFilesDownloadCourse_NoOverwrite(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/files", 200, []map[string]any{
		{
			"id":           "10",
			"display_name": "syllabus.pdf",
			"filename":     "syllabus.pdf",
			"size":         1024,
			"content_type": "application/pdf",
		},
	})

	outDir := t.TempDir()
	// Pre-create the file
	os.WriteFile(filepath.Join(outDir, "syllabus.pdf"), []byte("original"), 0644)

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "test-token",
		Profile: "default",
	}

	var buf bytes.Buffer
	cmd := newFilesDownloadCourseCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("out", outDir)
	_ = cmd.Flags().Set("no-overwrite", "true")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("files download-course --no-overwrite failed: %v", err)
	}

	// Verify file was not overwritten
	content, _ := os.ReadFile(filepath.Join(outDir, "syllabus.pdf"))
	if string(content) != "original" {
		t.Errorf("file was overwritten: got %q, want %q", content, "original")
	}
}
