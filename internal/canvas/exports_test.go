package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartEpubExport(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EpubExport{
			ID:            "42",
			CreatedAt:     "2026-06-15T10:00:00Z",
			WorkflowState: "created",
			ProgressURL:   "/api/v1/progress/1",
			UserID:        "5",
			Attachment: &Attachment{
				ID:       "99",
				Filename: "course.epub",
				URL:      "https://canvas.example.com/files/99/download",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	export, err := StartEpubExport(context.Background(), c, "10")
	if err != nil {
		t.Fatalf("StartEpubExport() error: %v", err)
	}

	wantPath := "/api/v1/courses/10/epub_exports"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want %q", gotMethod, "POST")
	}

	if export.ID != "42" {
		t.Errorf("export.ID = %q, want %q", export.ID, "42")
	}
	if export.WorkflowState != "created" {
		t.Errorf("export.WorkflowState = %q, want %q", export.WorkflowState, "created")
	}
	if export.ProgressURL != "/api/v1/progress/1" {
		t.Errorf("export.ProgressURL = %q, want %q", export.ProgressURL, "/api/v1/progress/1")
	}
	if export.Attachment == nil {
		t.Fatal("export.Attachment is nil, want non-nil")
	}
	if export.Attachment.Filename != "course.epub" {
		t.Errorf("export.Attachment.Filename = %q, want %q", export.Attachment.Filename, "course.epub")
	}
}

func TestStartEpubExportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"Unauthorized"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := StartEpubExport(context.Background(), c, "10")
	if err == nil {
		t.Fatal("StartEpubExport() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "start epub export for course 10") {
		t.Errorf("error = %q, want to contain 'start epub export for course 10'", err.Error())
	}
}

func TestGetEpubExport(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EpubExport{
			ID:            "42",
			WorkflowState: "exporting",
			ProgressURL:   "/api/v1/progress/1",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	export, err := GetEpubExport(context.Background(), c, "10", "42")
	if err != nil {
		t.Fatalf("GetEpubExport() error: %v", err)
	}

	wantPath := "/api/v1/courses/10/epub_exports/42"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want %q", gotMethod, "GET")
	}
	if export.ID != "42" {
		t.Errorf("export.ID = %q, want %q", export.ID, "42")
	}
	if export.WorkflowState != "exporting" {
		t.Errorf("export.WorkflowState = %q, want %q", export.WorkflowState, "exporting")
	}
}

func TestGetEpubExportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":[{"message":"Not Found"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GetEpubExport(context.Background(), c, "10", "999")
	if err == nil {
		t.Fatal("GetEpubExport() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get epub export 999") {
		t.Errorf("error = %q, want to contain 'get epub export 999'", err.Error())
	}
}

func TestListEpubExports(t *testing.T) {
	var gotPath string
	var gotPerPage string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<https://canvas.example.com/api/v1/epub_exports?page=1>; rel="current"`)
		json.NewEncoder(w).Encode([]CourseEpubExport{
			{
				ID:   "1",
				Name: "Math 101",
				EpubExport: &EpubExport{
					ID:            "10",
					WorkflowState: "completed",
				},
			},
			{
				ID:   "2",
				Name: "Science 200",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	exports, meta, err := ListEpubExports(context.Background(), c)
	if err != nil {
		t.Fatalf("ListEpubExports() error: %v", err)
	}

	wantPath := "/api/v1/epub_exports"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotPerPage != "100" {
		t.Errorf("per_page = %q, want %q", gotPerPage, "100")
	}

	if len(exports) != 2 {
		t.Fatalf("len(exports) = %d, want 2", len(exports))
	}
	if exports[0].ID != "1" {
		t.Errorf("exports[0].ID = %q, want %q", exports[0].ID, "1")
	}
	if exports[0].Name != "Math 101" {
		t.Errorf("exports[0].Name = %q, want %q", exports[0].Name, "Math 101")
	}
	if exports[0].EpubExport == nil {
		t.Fatal("exports[0].EpubExport is nil, want non-nil")
	}
	if exports[0].EpubExport.WorkflowState != "completed" {
		t.Errorf("exports[0].EpubExport.WorkflowState = %q, want %q", exports[0].EpubExport.WorkflowState, "completed")
	}
	if exports[1].EpubExport != nil {
		t.Errorf("exports[1].EpubExport = %v, want nil", exports[1].EpubExport)
	}
	_ = meta // PaginationMeta returned; basic presence check is enough
}

func TestListEpubExportsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":[{"message":"Internal Server Error"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, _, err := ListEpubExports(context.Background(), c)
	if err == nil {
		t.Fatal("ListEpubExports() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "list epub exports") {
		t.Errorf("error = %q, want to contain 'list epub exports'", err.Error())
	}
}

func TestStartContentExport(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotContentType string
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(r.Body)
		gotBody = string(bodyBytes)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContentExport{
			ID:            "55",
			ExportType:    "qti",
			WorkflowState: "created",
			ProgressURL:   "/api/v1/progress/2",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	export, err := StartContentExport(context.Background(), c, "10", "qti")
	if err != nil {
		t.Fatalf("StartContentExport() error: %v", err)
	}

	wantPath := "/api/v1/courses/10/content_exports"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want %q", gotMethod, "POST")
	}
	if !strings.Contains(gotContentType, "application/x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q, want to contain 'application/x-www-form-urlencoded'", gotContentType)
	}
	if !strings.Contains(gotBody, "export_type=qti") {
		t.Errorf("body = %q, want to contain 'export_type=qti'", gotBody)
	}

	if export.ID != "55" {
		t.Errorf("export.ID = %q, want %q", export.ID, "55")
	}
	if export.ExportType != "qti" {
		t.Errorf("export.ExportType = %q, want %q", export.ExportType, "qti")
	}
	if export.WorkflowState != "created" {
		t.Errorf("export.WorkflowState = %q, want %q", export.WorkflowState, "created")
	}
}

func TestStartContentExportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":[{"message":"Forbidden"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := StartContentExport(context.Background(), c, "10", "qti")
	if err == nil {
		t.Fatal("StartContentExport() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "start content export for course 10") {
		t.Errorf("error = %q, want to contain 'start content export for course 10'", err.Error())
	}
}

func TestGetContentExport(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContentExport{
			ID:            "55",
			ExportType:    "qti",
			WorkflowState: "exporting",
			Attachment: &Attachment{
				ID:       "77",
				Filename: "export.zip",
				URL:      "https://canvas.example.com/files/77/download",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	export, err := GetContentExport(context.Background(), c, "10", "55")
	if err != nil {
		t.Fatalf("GetContentExport() error: %v", err)
	}

	wantPath := "/api/v1/courses/10/content_exports/55"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want %q", gotMethod, "GET")
	}
	if export.ID != "55" {
		t.Errorf("export.ID = %q, want %q", export.ID, "55")
	}
	if export.ExportType != "qti" {
		t.Errorf("export.ExportType = %q, want %q", export.ExportType, "qti")
	}
	if export.Attachment == nil {
		t.Fatal("export.Attachment is nil, want non-nil")
	}
	if export.Attachment.Filename != "export.zip" {
		t.Errorf("export.Attachment.Filename = %q, want %q", export.Attachment.Filename, "export.zip")
	}
}

func TestGetContentExportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":[{"message":"Not Found"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GetContentExport(context.Background(), c, "10", "999")
	if err == nil {
		t.Fatal("GetContentExport() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get content export 999") {
		t.Errorf("error = %q, want to contain 'get content export 999'", err.Error())
	}
}

func TestListContentExports(t *testing.T) {
	var gotPath string
	var gotPerPage string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<https://canvas.example.com/api/v1/courses/10/content_exports?page=1>; rel="current"`)
		json.NewEncoder(w).Encode([]ContentExport{
			{
				ID:            "1",
				ExportType:    "qti",
				WorkflowState: "completed",
			},
			{
				ID:            "2",
				ExportType:    "common_cartridge",
				WorkflowState: "exporting",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	exports, meta, err := ListContentExports(context.Background(), c, "10")
	if err != nil {
		t.Fatalf("ListContentExports() error: %v", err)
	}

	wantPath := "/api/v1/courses/10/content_exports"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotPerPage != "100" {
		t.Errorf("per_page = %q, want %q", gotPerPage, "100")
	}

	if len(exports) != 2 {
		t.Fatalf("len(exports) = %d, want 2", len(exports))
	}
	if exports[0].ID != "1" {
		t.Errorf("exports[0].ID = %q, want %q", exports[0].ID, "1")
	}
	if exports[0].ExportType != "qti" {
		t.Errorf("exports[0].ExportType = %q, want %q", exports[0].ExportType, "qti")
	}
	if exports[1].ExportType != "common_cartridge" {
		t.Errorf("exports[1].ExportType = %q, want %q", exports[1].ExportType, "common_cartridge")
	}
	_ = meta
}

func TestListContentExportsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":[{"message":"Internal Server Error"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, _, err := ListContentExports(context.Background(), c, "10")
	if err == nil {
		t.Fatal("ListContentExports() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "list content exports for course 10") {
		t.Errorf("error = %q, want to contain 'list content exports for course 10'", err.Error())
	}
}

func TestGetProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		completion := 75.0
		json.NewEncoder(w).Encode(Progress{
			ID:            "1",
			ContextID:     "10",
			ContextType:   "Course",
			UserID:        "5",
			Tag:           "epub_export",
			Completion:    &completion,
			WorkflowState: "running",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	progressURL := srv.URL + "/api/v1/progress/1"
	progress, err := GetProgress(context.Background(), c, progressURL)
	if err != nil {
		t.Fatalf("GetProgress() error: %v", err)
	}

	if progress.ID != "1" {
		t.Errorf("progress.ID = %q, want %q", progress.ID, "1")
	}
	if progress.Tag != "epub_export" {
		t.Errorf("progress.Tag = %q, want %q", progress.Tag, "epub_export")
	}
	if progress.Completion == nil {
		t.Fatal("progress.Completion is nil, want non-nil")
	}
	if *progress.Completion != 75.0 {
		t.Errorf("progress.Completion = %f, want 75.0", *progress.Completion)
	}
	if progress.WorkflowState != "running" {
		t.Errorf("progress.WorkflowState = %q, want %q", progress.WorkflowState, "running")
	}
}

func TestGetProgressError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":[{"message":"Not Found"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	progressURL := srv.URL + "/api/v1/progress/999"
	_, err := GetProgress(context.Background(), c, progressURL)
	if err == nil {
		t.Fatal("GetProgress() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get progress") {
		t.Errorf("error = %q, want to contain 'get progress'", err.Error())
	}
}

func TestGetProgressInvalidURL(t *testing.T) {
	c := NewClient("http://localhost", "tok", "0.1.0", 5*time.Second, 0)

	// URL with invalid characters that url.Parse will reject
	_, err := GetProgress(context.Background(), c, "http://localhost/\x00progress")
	if err == nil {
		t.Fatal("GetProgress() expected error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "parse progress URL") {
		t.Errorf("error = %q, want to contain 'parse progress URL'", err.Error())
	}
}

func TestDownloadExport(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/zip")
		w.Write([]byte("PK\x03\x04fake-zip-content"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var buf bytes.Buffer
	attachmentURL := srv.URL + "/files/99/download"
	err := DownloadExport(context.Background(), c, attachmentURL, &buf)
	if err != nil {
		t.Fatalf("DownloadExport() error: %v", err)
	}

	if gotPath != "/files/99/download" {
		t.Errorf("path = %q, want %q", gotPath, "/files/99/download")
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want %q", gotMethod, "GET")
	}
	if buf.String() != "PK\x03\x04fake-zip-content" {
		t.Errorf("body = %q, want %q", buf.String(), "PK\x03\x04fake-zip-content")
	}
}

func TestDownloadExport4xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var buf bytes.Buffer
	attachmentURL := srv.URL + "/files/99/download"
	err := DownloadExport(context.Background(), c, attachmentURL, &buf)
	if err == nil {
		t.Fatal("DownloadExport() expected error for 4xx, got nil")
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Errorf("error = %q, want to contain 'status 403'", err.Error())
	}
}

func TestDownloadExportInvalidURL(t *testing.T) {
	c := NewClient("http://localhost", "tok", "0.1.0", 5*time.Second, 0)

	var buf bytes.Buffer
	err := DownloadExport(context.Background(), c, "http://localhost/\x00file", &buf)
	if err == nil {
		t.Fatal("DownloadExport() expected error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "parse attachment URL") {
		t.Errorf("error = %q, want to contain 'parse attachment URL'", err.Error())
	}
}

func TestWaitForExportCompleted(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		state := "running"
		if n >= 2 {
			state = "completed"
		}
		json.NewEncoder(w).Encode(Progress{
			ID:            "1",
			WorkflowState: state,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	progressURL := srv.URL + "/api/v1/progress/1"
	err := WaitForExport(context.Background(), c, progressURL, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForExport() error: %v", err)
	}

	if n := callCount.Load(); n < 2 {
		t.Errorf("poll count = %d, want at least 2", n)
	}
}

func TestWaitForExportFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Progress{
			ID:            "1",
			WorkflowState: "failed",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	progressURL := srv.URL + "/api/v1/progress/1"
	err := WaitForExport(context.Background(), c, progressURL, 10*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForExport() expected error for failed state, got nil")
	}
	if !strings.Contains(err.Error(), "export failed") {
		t.Errorf("error = %q, want to contain 'export failed'", err.Error())
	}
}

func TestWaitForExportContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Progress{
			ID:            "1",
			WorkflowState: "running",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a very short time so the poll loop hits ctx.Done()
	timer := time.AfterFunc(15*time.Millisecond, cancel)
	defer timer.Stop()

	progressURL := srv.URL + "/api/v1/progress/1"
	err := WaitForExport(ctx, c, progressURL, 10*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForExport() expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestWaitForExportProgressError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":[{"message":"Internal Server Error"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	progressURL := srv.URL + "/api/v1/progress/1"
	err := WaitForExport(context.Background(), c, progressURL, 10*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForExport() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "get progress") {
		t.Errorf("error = %q, want to contain 'get progress'", err.Error())
	}
}

func TestFormReader(t *testing.T) {
	values := url.Values{}
	values.Set("export_type", "qti")
	values.Set("skip", "attachments")

	reader := formReader(values)
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	decoded, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatalf("ParseQuery() error: %v", err)
	}

	if decoded.Get("export_type") != "qti" {
		t.Errorf("export_type = %q, want %q", decoded.Get("export_type"), "qti")
	}
	if decoded.Get("skip") != "attachments" {
		t.Errorf("skip = %q, want %q", decoded.Get("skip"), "attachments")
	}
}

func TestFormReaderEmpty(t *testing.T) {
	values := url.Values{}
	reader := formReader(values)
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if string(body) != "" {
		t.Errorf("body = %q, want empty string", string(body))
	}
}
