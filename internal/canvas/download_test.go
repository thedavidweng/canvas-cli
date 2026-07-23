package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadCourseFiles_Success(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/courses/42/files":
			json.NewEncoder(w).Encode([]File{
				{ID: "101", Filename: "a.txt", DisplayName: "a.txt", Size: 5, ContentType: "text/plain"},
				{ID: "102", Filename: "b.txt", DisplayName: "b.txt", Size: 5, ContentType: "text/plain"},
			})
		case "/api/v1/files/101":
			fmt.Fprintf(w, `{"id":"101","url":"%s/raw/101","size":5}`, srv.URL)
		case "/api/v1/files/102":
			fmt.Fprintf(w, `{"id":"102","url":"%s/raw/102","size":5}`, srv.URL)
		case "/raw/101":
			w.Write([]byte("hello"))
		case "/raw/102":
			w.Write([]byte("world"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	tmpDir := t.TempDir()
	result, err := DownloadCourseFiles(context.Background(), c, DownloadCourseFilesOptions{
		CourseID: "42",
		OutDir:   tmpDir,
	})
	if err != nil {
		t.Fatalf("DownloadCourseFiles() error: %v", err)
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

	// Verify files were written.
	for _, name := range []string{"a.txt", "b.txt"} {
		data, err := os.ReadFile(filepath.Join(tmpDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if len(data) != 5 {
			t.Errorf("%s content length = %d, want 5", name, len(data))
		}
	}

	// Verify manifest.json.
	manifestData, err := os.ReadFile(filepath.Join(tmpDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	var entries []FileManifestEntry
	if err := json.Unmarshal(manifestData, &entries); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("manifest entries = %d, want 2", len(entries))
	}
	if entries[0].DownloadStatus != "ok" {
		t.Errorf("entries[0].DownloadStatus = %q, want ok", entries[0].DownloadStatus)
	}

	// Verify manifest.ndjson.
	ndjsonData, err := os.ReadFile(filepath.Join(tmpDir, "manifest.ndjson"))
	if err != nil {
		t.Fatalf("read manifest.ndjson: %v", err)
	}
	if len(ndjsonData) == 0 {
		t.Error("manifest.ndjson should not be empty")
	}

	if result.ManifestPath != filepath.Join(tmpDir, "manifest.json") {
		t.Errorf("ManifestPath = %q, want %q", result.ManifestPath, filepath.Join(tmpDir, "manifest.json"))
	}
}

func TestDownloadCourseFiles_NoOverwrite_SkipsExisting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/courses/42/files":
			json.NewEncoder(w).Encode([]File{
				{ID: "101", Filename: "a.txt", DisplayName: "a.txt", Size: 5, ContentType: "text/plain"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	tmpDir := t.TempDir()
	// Pre-create the file so it should be skipped.
	existingPath := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(existingPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := DownloadCourseFiles(context.Background(), c, DownloadCourseFilesOptions{
		CourseID:    "42",
		OutDir:      tmpDir,
		NoOverwrite: true,
	})
	if err != nil {
		t.Fatalf("DownloadCourseFiles() error: %v", err)
	}

	if result.Downloaded != 1 {
		t.Errorf("Downloaded = %d, want 1 (skipped counts as downloaded)", result.Downloaded)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("Entries = %d, want 1", len(result.Entries))
	}
	if result.Entries[0].DownloadStatus != "skipped" {
		t.Errorf("DownloadStatus = %q, want skipped", result.Entries[0].DownloadStatus)
	}

	// File should not have been overwritten.
	data, _ := os.ReadFile(existingPath)
	if string(data) != "old" {
		t.Errorf("file was overwritten, content = %q, want %q", string(data), "old")
	}
}

func TestDownloadCourseFiles_DownloadError(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/courses/42/files":
			json.NewEncoder(w).Encode([]File{
				{ID: "101", Filename: "a.txt", DisplayName: "a.txt", Size: 5, ContentType: "text/plain"},
			})
		case "/api/v1/files/101":
			fmt.Fprintf(w, `{"id":"101","url":"%s/raw/101","size":5}`, srv.URL)
		case "/raw/101":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"download failed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	tmpDir := t.TempDir()
	result, err := DownloadCourseFiles(context.Background(), c, DownloadCourseFilesOptions{
		CourseID: "42",
		OutDir:   tmpDir,
	})
	if err != nil {
		t.Fatalf("DownloadCourseFiles() error: %v", err)
	}

	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
	if result.Downloaded != 0 {
		t.Errorf("Downloaded = %d, want 0", result.Downloaded)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("Entries = %d, want 1", len(result.Entries))
	}
	if result.Entries[0].DownloadStatus != "error" {
		t.Errorf("DownloadStatus = %q, want error", result.Entries[0].DownloadStatus)
	}
	if result.Entries[0].Error == "" {
		t.Error("Error should not be empty")
	}
}

func TestDownloadCourseFiles_ListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"server error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	tmpDir := t.TempDir()
	_, err := DownloadCourseFiles(context.Background(), c, DownloadCourseFilesOptions{
		CourseID: "42",
		OutDir:   tmpDir,
	})
	if err == nil {
		t.Fatal("expected error for list failure, got nil")
	}
}

func TestDownloadCourseFiles_EmptyFileList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]File{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	tmpDir := t.TempDir()
	result, err := DownloadCourseFiles(context.Background(), c, DownloadCourseFilesOptions{
		CourseID: "42",
		OutDir:   tmpDir,
	})
	if err != nil {
		t.Fatalf("DownloadCourseFiles() error: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if result.Downloaded != 0 {
		t.Errorf("Downloaded = %d, want 0", result.Downloaded)
	}
}

func TestDownloadCourseFiles_PathTraversalSanitized(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/courses/42/files":
			json.NewEncoder(w).Encode([]File{
				{ID: "101", Filename: "../../.bashrc", DisplayName: "evil", Size: 5, ContentType: "text/plain"},
				{ID: "102", Filename: "/etc/passwd", DisplayName: "absolute", Size: 5, ContentType: "text/plain"},
			})
		case "/api/v1/files/101":
			fmt.Fprintf(w, `{"id":"101","url":"%s/raw/101","size":5}`, srv.URL)
		case "/api/v1/files/102":
			fmt.Fprintf(w, `{"id":"102","url":"%s/raw/102","size":5}`, srv.URL)
		case "/raw/101":
			w.Write([]byte("hello"))
		case "/raw/102":
			w.Write([]byte("world"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	tmpDir := t.TempDir()
	result, err := DownloadCourseFiles(context.Background(), c, DownloadCourseFilesOptions{
		CourseID: "42",
		OutDir:   tmpDir,
	})
	if err != nil {
		t.Fatalf("DownloadCourseFiles() error: %v", err)
	}

	if result.Downloaded != 2 {
		t.Fatalf("Downloaded = %d, want 2", result.Downloaded)
	}

	// Files should be written inside tmpDir with sanitized names, not outside.
	for _, name := range []string{".bashrc", "passwd"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected sanitized file %s to exist: %v", path, err)
		}
	}

	// Verify no file was created outside tmpDir.
	outsidePath := filepath.Join(filepath.Dir(tmpDir), ".bashrc")
	if _, err := os.Stat(outsidePath); err == nil {
		t.Errorf("traversal file should not exist at %s", outsidePath)
	}
}
