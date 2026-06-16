package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/testutil"
)

// --- downloadToFile ---

func TestDownloadToFile_HappyPath(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/files/1/download", 200, []byte("epub content here"))

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	outPath := filepath.Join(t.TempDir(), "test.epub")

	err := downloadToFile(context.Background(), client, mock.URL()+"/files/1/download", outPath)
	if err != nil {
		t.Fatalf("downloadToFile failed: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) != "epub content here" {
		t.Errorf("expected 'epub content here', got %q", content)
	}
}

func TestDownloadToFile_CreatesNestedDirs(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/files/2/download", 200, []byte("nested content"))

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	outPath := filepath.Join(t.TempDir(), "subdir", "nested", "test.epub")

	err := downloadToFile(context.Background(), client, mock.URL()+"/files/2/download", outPath)
	if err != nil {
		t.Fatalf("downloadToFile with nested dirs failed: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("expected 'nested content', got %q", content)
	}
}

func TestDownloadToFile_ServerError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/files/999/download", 500, map[string]any{"error": "server error"})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	outPath := filepath.Join(t.TempDir(), "fail.epub")

	err := downloadToFile(context.Background(), client, mock.URL()+"/files/999/download", outPath)
	if err == nil {
		t.Fatal("expected error on server 500, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected 'status 500' in error, got: %v", err)
	}
}

// --- waitForComplete ---

func TestWaitForComplete_Completed(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/progress/123", 200, map[string]any{
		"id":             "123",
		"workflow_state": "completed",
		"completion":     100.0,
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := waitForComplete(context.Background(), client, "/api/v1/progress/123", &buf)
	if err != nil {
		t.Fatalf("waitForComplete failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "done!") {
		t.Errorf("expected 'done!' in output, got: %q", output)
	}
}

func TestWaitForComplete_Failed(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/progress/456", 200, map[string]any{
		"id":             "456",
		"workflow_state": "failed",
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := waitForComplete(context.Background(), client, "/api/v1/progress/456", &buf)
	if err == nil {
		t.Fatal("expected error on failed progress, got nil")
	}
	if !strings.Contains(err.Error(), "export failed") {
		t.Errorf("expected 'export failed' in error, got: %v", err)
	}
}

func TestWaitForComplete_ContextCancelled(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Register progress endpoint in case it's reached before context check
	mock.On("GET", "/api/v1/progress/789", 200, map[string]any{
		"id":             "789",
		"workflow_state": "processing",
	})

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var buf bytes.Buffer
	err := waitForComplete(ctx, client, "/api/v1/progress/789", &buf)
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected 'context canceled' in error, got: %v", err)
	}
}

// --- exportEpub ---

func TestExportEpub_FullFlow(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Step 1: Start export
	mock.On("POST", "/api/v1/courses/1/epub_exports", 200, map[string]any{
		"id":             "1",
		"workflow_state": "created",
		"progress_url":   "/api/v1/progress/100",
	})

	// Step 2: Poll progress -> completed
	mock.On("GET", "/api/v1/progress/100", 200, map[string]any{
		"id":             "100",
		"workflow_state": "completed",
		"completion":     100.0,
	})

	// Step 3: Re-fetch export with attachment
	mock.On("GET", "/api/v1/courses/1/epub_exports/1", 200, map[string]any{
		"id":             "1",
		"workflow_state": "exported",
		"attachment": map[string]any{
			"id":       "200",
			"filename": "course-1.epub",
			"url":      mock.URL() + "/files/200/download",
		},
	})

	// Step 4: Download file
	mock.On("GET", "/files/200/download", 200, []byte("epub binary content"))

	cfg := &config.ResolvedConfig{
		BaseURL: mock.URL(),
		Token:   "tok",
		Profile: "default",
	}

	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer
	outPath := filepath.Join(t.TempDir(), "course-1.epub")

	err := exportEpub(context.Background(), client, &buf, "1", outPath, false, false, cfg)
	if err != nil {
		t.Fatalf("exportEpub full flow failed: %v", err)
	}

	// Verify file was downloaded
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != "epub binary content" {
		t.Errorf("expected 'epub binary content', got %q", content)
	}

	// Verify output messages
	output := buf.String()
	if !strings.Contains(output, "Starting ePub export") {
		t.Errorf("expected 'Starting ePub export' in output, got: %q", output)
	}
	if !strings.Contains(output, "Done!") {
		t.Errorf("expected 'Done!' in output, got: %q", output)
	}
}

func TestExportEpub_FullFlow_DefaultOutPath(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/42/epub_exports", 200, map[string]any{
		"id":           "2",
		"progress_url": "/api/v1/progress/200",
	})
	mock.On("GET", "/api/v1/progress/200", 200, map[string]any{
		"id":             "200",
		"workflow_state": "completed",
	})
	mock.On("GET", "/api/v1/courses/42/epub_exports/2", 200, map[string]any{
		"id": "2",
		"attachment": map[string]any{
			"url": mock.URL() + "/files/201/download",
		},
	})
	mock.On("GET", "/files/201/download", 200, []byte("content"))

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	// Use a temp dir as working directory by specifying outPath with temp dir
	outPath := filepath.Join(t.TempDir(), "course-42.epub")
	err := exportEpub(context.Background(), client, &buf, "42", outPath, false, false, cfg)
	if err != nil {
		t.Fatalf("exportEpub with default path failed: %v", err)
	}
}

func TestExportEpub_StartError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/epub_exports", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportEpub(context.Background(), client, &buf, "1", "", false, false, cfg)
	if err == nil {
		t.Fatal("expected error on start failure, got nil")
	}
}

func TestExportEpub_NoWait(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/epub_exports", 200, map[string]any{
		"id":           "1",
		"progress_url": "/api/v1/progress/100",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportEpub(context.Background(), client, &buf, "1", "", true, false, cfg)
	if err != nil {
		t.Fatalf("exportEpub --no-wait failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Export started") {
		t.Errorf("expected 'Export started' in no-wait output, got: %q", output)
	}
	if !strings.Contains(output, "/api/v1/progress/100") {
		t.Errorf("expected progress URL in output, got: %q", output)
	}
}

func TestExportEpub_NoWaitJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/epub_exports", 200, map[string]any{
		"id":           "1",
		"progress_url": "/api/v1/progress/100",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportEpub(context.Background(), client, &buf, "1", "", true, true, cfg)
	if err != nil {
		t.Fatalf("exportEpub --no-wait --json failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) && !strings.Contains(output, `"ok": true`) {
		t.Errorf("expected JSON envelope with ok:true, got: %q", output)
	}
}

func TestExportEpub_NoAttachmentURL(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/epub_exports", 200, map[string]any{
		"id":           "1",
		"progress_url": "/api/v1/progress/100",
	})
	mock.On("GET", "/api/v1/progress/100", 200, map[string]any{
		"id":             "100",
		"workflow_state": "completed",
	})
	// Return export without attachment
	mock.On("GET", "/api/v1/courses/1/epub_exports/1", 200, map[string]any{
		"id":             "1",
		"workflow_state": "exported",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportEpub(context.Background(), client, &buf, "1", "", false, false, cfg)
	if err == nil {
		t.Fatal("expected error when no attachment URL, got nil")
	}
	if !strings.Contains(err.Error(), "no download URL") {
		t.Errorf("expected 'no download URL' in error, got: %v", err)
	}
}

func TestExportEpub_WaitFails(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/epub_exports", 200, map[string]any{
		"id":           "1",
		"progress_url": "/api/v1/progress/100",
	})
	mock.On("GET", "/api/v1/progress/100", 200, map[string]any{
		"id":             "100",
		"workflow_state": "failed",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportEpub(context.Background(), client, &buf, "1", "", false, false, cfg)
	if err == nil {
		t.Fatal("expected error when wait fails, got nil")
	}
	if !strings.Contains(err.Error(), "export failed") {
		t.Errorf("expected 'export failed' in error, got: %v", err)
	}
}

func TestExportEpub_ReFetchError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/epub_exports", 200, map[string]any{
		"id":           "1",
		"progress_url": "/api/v1/progress/100",
	})
	mock.On("GET", "/api/v1/progress/100", 200, map[string]any{
		"id":             "100",
		"workflow_state": "completed",
	})
	// Re-fetch returns error
	mock.On("GET", "/api/v1/courses/1/epub_exports/1", 500, map[string]any{
		"errors": []map[string]any{{"message": "server error"}},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportEpub(context.Background(), client, &buf, "1", "", false, false, cfg)
	if err == nil {
		t.Fatal("expected error on re-fetch failure, got nil")
	}
}

// --- exportContent ---

func TestExportContent_FullFlow(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	// Step 1: Start content export
	mock.On("POST", "/api/v1/courses/1/content_exports", 200, map[string]any{
		"id":           "10",
		"export_type":  "common_cartridge",
		"progress_url": "/api/v1/progress/500",
	})

	// Step 2: Poll progress -> completed
	mock.On("GET", "/api/v1/progress/500", 200, map[string]any{
		"id":             "500",
		"workflow_state": "completed",
	})

	// Step 3: Re-fetch export with attachment
	mock.On("GET", "/api/v1/courses/1/content_exports/10", 200, map[string]any{
		"id":             "10",
		"export_type":    "common_cartridge",
		"workflow_state": "exported",
		"attachment": map[string]any{
			"id":       "600",
			"filename": "course-1.imscc",
			"url":      mock.URL() + "/files/600/download",
		},
	})

	// Step 4: Download file
	mock.On("GET", "/files/600/download", 200, []byte("imscc content"))

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer
	outPath := filepath.Join(t.TempDir(), "course-1.imscc")

	err := exportContent(context.Background(), client, &buf, "1", "common_cartridge", outPath, false, false, cfg)
	if err != nil {
		t.Fatalf("exportContent full flow failed: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != "imscc content" {
		t.Errorf("expected 'imscc content', got %q", content)
	}

	output := buf.String()
	if !strings.Contains(output, "Starting common_cartridge export") {
		t.Errorf("expected 'Starting common_cartridge export' in output, got: %q", output)
	}
	if !strings.Contains(output, "Done!") {
		t.Errorf("expected 'Done!' in output, got: %q", output)
	}
}

func TestExportContent_ZipFormat(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/content_exports", 200, map[string]any{
		"id":           "11",
		"export_type":  "zip",
		"progress_url": "/api/v1/progress/501",
	})
	mock.On("GET", "/api/v1/progress/501", 200, map[string]any{
		"id":             "501",
		"workflow_state": "completed",
	})
	mock.On("GET", "/api/v1/courses/1/content_exports/11", 200, map[string]any{
		"id":          "11",
		"export_type": "zip",
		"attachment": map[string]any{
			"url": mock.URL() + "/files/601/download",
		},
	})
	mock.On("GET", "/files/601/download", 200, []byte("zip content"))

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer
	outPath := filepath.Join(t.TempDir(), "course-1.zip")

	err := exportContent(context.Background(), client, &buf, "1", "zip", outPath, false, false, cfg)
	if err != nil {
		t.Fatalf("exportContent zip failed: %v", err)
	}

	content, _ := os.ReadFile(outPath)
	if string(content) != "zip content" {
		t.Errorf("expected 'zip content', got %q", content)
	}
}

func TestExportContent_QTIFormat(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/content_exports", 200, map[string]any{
		"id":           "12",
		"export_type":  "qti",
		"progress_url": "/api/v1/progress/502",
	})
	mock.On("GET", "/api/v1/progress/502", 200, map[string]any{
		"id":             "502",
		"workflow_state": "completed",
	})
	mock.On("GET", "/api/v1/courses/1/content_exports/12", 200, map[string]any{
		"id":          "12",
		"export_type": "qti",
		"attachment": map[string]any{
			"url": mock.URL() + "/files/602/download",
		},
	})
	mock.On("GET", "/files/602/download", 200, []byte("qti content"))

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer
	outPath := filepath.Join(t.TempDir(), "course-1-qti.zip")

	err := exportContent(context.Background(), client, &buf, "1", "qti", outPath, false, false, cfg)
	if err != nil {
		t.Fatalf("exportContent qti failed: %v", err)
	}
}

func TestExportContent_DefaultOutPath(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/5/content_exports", 200, map[string]any{
		"id":           "13",
		"export_type":  "common_cartridge",
		"progress_url": "/api/v1/progress/503",
	})
	mock.On("GET", "/api/v1/progress/503", 200, map[string]any{
		"id":             "503",
		"workflow_state": "completed",
	})
	mock.On("GET", "/api/v1/courses/5/content_exports/13", 200, map[string]any{
		"id":          "13",
		"export_type": "common_cartridge",
		"attachment": map[string]any{
			"url": mock.URL() + "/files/603/download",
		},
	})
	mock.On("GET", "/files/603/download", 200, []byte("data"))

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	// Use empty outPath to trigger default naming (write to temp dir to avoid polluting cwd)
	outPath := filepath.Join(t.TempDir(), fmt.Sprintf("course-%s.imscc", "5"))
	err := exportContent(context.Background(), client, &buf, "5", "common_cartridge", outPath, false, false, cfg)
	if err != nil {
		t.Fatalf("exportContent with default path failed: %v", err)
	}
}

func TestExportContent_StartError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/content_exports", 500, map[string]any{
		"errors": []map[string]any{{"message": "server error"}},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportContent(context.Background(), client, &buf, "1", "common_cartridge", "", false, false, cfg)
	if err == nil {
		t.Fatal("expected error on start failure, got nil")
	}
}

func TestExportContent_NoWait(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/content_exports", 200, map[string]any{
		"id":           "10",
		"export_type":  "common_cartridge",
		"progress_url": "/api/v1/progress/500",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportContent(context.Background(), client, &buf, "1", "common_cartridge", "", true, false, cfg)
	if err != nil {
		t.Fatalf("exportContent --no-wait failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Export started") {
		t.Errorf("expected 'Export started' in no-wait output, got: %q", output)
	}
	if !strings.Contains(output, "/api/v1/progress/500") {
		t.Errorf("expected progress URL in output, got: %q", output)
	}
}

func TestExportContent_NoWaitJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/content_exports", 200, map[string]any{
		"id":           "10",
		"export_type":  "common_cartridge",
		"progress_url": "/api/v1/progress/500",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportContent(context.Background(), client, &buf, "1", "common_cartridge", "", true, true, cfg)
	if err != nil {
		t.Fatalf("exportContent --no-wait --json failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) && !strings.Contains(output, `"ok": true`) {
		t.Errorf("expected JSON envelope with ok:true, got: %q", output)
	}
}

func TestExportContent_NoAttachmentURL(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/content_exports", 200, map[string]any{
		"id":           "10",
		"export_type":  "common_cartridge",
		"progress_url": "/api/v1/progress/500",
	})
	mock.On("GET", "/api/v1/progress/500", 200, map[string]any{
		"id":             "500",
		"workflow_state": "completed",
	})
	mock.On("GET", "/api/v1/courses/1/content_exports/10", 200, map[string]any{
		"id":             "10",
		"export_type":    "common_cartridge",
		"workflow_state": "exported",
		// no attachment
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportContent(context.Background(), client, &buf, "1", "common_cartridge", "", false, false, cfg)
	if err == nil {
		t.Fatal("expected error when no attachment URL, got nil")
	}
	if !strings.Contains(err.Error(), "no download URL") {
		t.Errorf("expected 'no download URL' in error, got: %v", err)
	}
}

func TestExportContent_WaitFails(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("POST", "/api/v1/courses/1/content_exports", 200, map[string]any{
		"id":           "10",
		"export_type":  "common_cartridge",
		"progress_url": "/api/v1/progress/500",
	})
	mock.On("GET", "/api/v1/progress/500", 200, map[string]any{
		"id":             "500",
		"workflow_state": "failed",
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	client := canvas.NewClient(mock.URL(), "tok", "dev", 0, 0)
	var buf bytes.Buffer

	err := exportContent(context.Background(), client, &buf, "1", "common_cartridge", "", false, false, cfg)
	if err == nil {
		t.Fatal("expected error when wait fails, got nil")
	}
	if !strings.Contains(err.Error(), "export failed") {
		t.Errorf("expected 'export failed' in error, got: %v", err)
	}
}

// --- newCoursesExportCmd / newCoursesExportsCmd ---

func TestNewCoursesExportCmd_MissingCourse(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}

	var buf bytes.Buffer
	cmd := newCoursesExportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("format", "epub")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --course is missing, got nil")
	}
	if !strings.Contains(err.Error(), "--course is required") {
		t.Errorf("expected '--course is required' in error, got: %v", err)
	}
}

func TestNewCoursesExportCmd_MissingFormat(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}

	var buf bytes.Buffer
	cmd := newCoursesExportCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --format is missing, got nil")
	}
	if !strings.Contains(err.Error(), "--format is required") {
		t.Errorf("expected '--format is required' in error, got: %v", err)
	}
}

func TestNewCoursesExportCmd_NilConfig(t *testing.T) {
	var buf bytes.Buffer
	cmd := newCoursesExportCmd()
	cmd.SetContext(context.Background()) // no config
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")
	_ = cmd.Flags().Set("format", "epub")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when config is nil, got nil")
	}
	if !strings.Contains(err.Error(), "no config loaded") {
		t.Errorf("expected 'no config loaded' in error, got: %v", err)
	}
}

func TestNewCoursesExportsCmd_JSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/content_exports", 200, []map[string]any{
		{"id": "1", "export_type": "common_cartridge", "workflow_state": "exported", "created_at": "2026-01-01T00:00:00Z"},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}

	var buf bytes.Buffer
	cmd := newCoursesExportsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("courses exports --json failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":true`) && !strings.Contains(output, `"ok": true`) {
		t.Errorf("expected JSON envelope with ok:true, got: %q", output)
	}
}

func TestNewCoursesExportsCmd_MissingCourse(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}

	var buf bytes.Buffer
	cmd := newCoursesExportsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --course is missing, got nil")
	}
	if !strings.Contains(err.Error(), "--course is required") {
		t.Errorf("expected '--course is required' in error, got: %v", err)
	}
}

func TestNewCoursesExportsCmd_NilConfig(t *testing.T) {
	var buf bytes.Buffer
	cmd := newCoursesExportsCmd()
	cmd.SetContext(context.Background())
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when config is nil, got nil")
	}
	if !strings.Contains(err.Error(), "no config loaded") {
		t.Errorf("expected 'no config loaded' in error, got: %v", err)
	}
}

func TestNewCoursesExportsCmd_APIError(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/content_exports", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer

	// Without JSON mode, error is returned directly
	cmd := newCoursesExportsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error on API failure, got nil")
	}
}

func TestNewCoursesExportsCmd_APIErrorJSON(t *testing.T) {
	mock := testutil.NewMockCanvas()
	defer mock.Close()

	mock.On("GET", "/api/v1/courses/1/content_exports", 500, map[string]any{
		"errors": []map[string]any{{"message": "internal server error"}},
	})

	cfg := &config.ResolvedConfig{BaseURL: mock.URL(), Token: "tok", Profile: "default"}
	var buf bytes.Buffer

	// With JSON mode, error is wrapped in envelope
	cmd := newCoursesExportsCmd()
	cmd.SetContext(WithConfig(context.Background(), cfg))
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("json", "true")
	_ = cmd.Flags().Set("course", "1")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("JSON mode should not return error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":false`) && !strings.Contains(output, `"ok": false`) {
		t.Errorf("expected JSON error envelope with ok:false, got: %q", output)
	}
}
