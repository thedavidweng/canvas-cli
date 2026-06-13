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
