package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestListFiles(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]File{
			{
				ID:          "101",
				DisplayName: "syllabus.pdf",
				Filename:    "syllabus.pdf",
				Size:        2048,
				ContentType: "application/pdf",
			},
			{
				ID:          "102",
				DisplayName: "slides.pptx",
				Filename:    "slides.pptx",
				Size:        4096,
				ContentType: "application/vnd.ms-powerpoint",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	opts := url.Values{}
	files, meta, err := ListFiles(context.Background(), c, "42", opts)
	if err != nil {
		t.Fatalf("ListFiles() error: %v", err)
	}

	wantPath := "/api/v1/courses/42/files"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if gotQuery.Get("per_page") != "100" {
		t.Errorf("per_page = %q, want %q", gotQuery.Get("per_page"), "100")
	}

	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].ID != "101" {
		t.Errorf("files[0].ID = %q, want %q", files[0].ID, "101")
	}
	if files[0].DisplayName != "syllabus.pdf" {
		t.Errorf("files[0].DisplayName = %q, want %q", files[0].DisplayName, "syllabus.pdf")
	}
	if files[1].ID != "102" {
		t.Errorf("files[1].ID = %q, want %q", files[1].ID, "102")
	}

	if meta.TotalItems != 2 {
		t.Errorf("meta.TotalItems = %d, want 2", meta.TotalItems)
	}
	if meta.RequestCount != 1 {
		t.Errorf("meta.RequestCount = %d, want 1", meta.RequestCount)
	}
}

func TestListFilesPagination(t *testing.T) {
	page := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/courses/42/files?page=2>; rel="next"`, srv.URL))
			json.NewEncoder(w).Encode([]File{
				{ID: "1", DisplayName: "a.pdf"},
				{ID: "2", DisplayName: "b.pdf"},
			})
		case 2:
			json.NewEncoder(w).Encode([]File{
				{ID: "3", DisplayName: "c.pdf"},
			})
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	opts := url.Values{}
	files, meta, err := ListFiles(context.Background(), c, "42", opts)
	if err != nil {
		t.Fatalf("ListFiles() error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("len(files) = %d, want 3", len(files))
	}
	if meta.RequestCount != 2 {
		t.Errorf("meta.RequestCount = %d, want 2", meta.RequestCount)
	}
	if meta.TotalItems != 3 {
		t.Errorf("meta.TotalItems = %d, want 3", meta.TotalItems)
	}
}

func TestGetFile(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(File{
			ID:          "55",
			DisplayName: "syllabus.pdf",
			Filename:    "syllabus.pdf",
			Size:        2048,
			ContentType: "application/pdf",
			URL:         "https://example.com/files/syllabus.pdf",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	file, err := GetFile(context.Background(), c, "55")
	if err != nil {
		t.Fatalf("GetFile() error: %v", err)
	}

	wantPath := "/api/v1/files/55"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	if file.ID != "55" {
		t.Errorf("file.ID = %q, want %q", file.ID, "55")
	}
	if file.DisplayName != "syllabus.pdf" {
		t.Errorf("file.DisplayName = %q, want %q", file.DisplayName, "syllabus.pdf")
	}
	if file.Size != 2048 {
		t.Errorf("file.Size = %d, want %d", file.Size, 2048)
	}
	if file.ContentType != "application/pdf" {
		t.Errorf("file.ContentType = %q, want %q", file.ContentType, "application/pdf")
	}
}

func TestGetFileNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := GetFile(context.Background(), c, "999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestDownloadFile(t *testing.T) {
	var gotPaths []string

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/files/55":
			// Return file metadata with download URL
			fmt.Fprintf(w, `{"id":"55","display_name":"report.pdf","url":"%s/files/report.pdf","size":1024}`, srv.URL)
		case "/files/report.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("file-content-here"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	var buf bytes.Buffer
	err := DownloadFile(context.Background(), c, "55", &buf)
	if err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}

	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(gotPaths))
	}
	if gotPaths[0] != "/api/v1/files/55" {
		t.Errorf("first request path = %q, want %q", gotPaths[0], "/api/v1/files/55")
	}
	if gotPaths[1] != "/files/report.pdf" {
		t.Errorf("second request path = %q, want %q", gotPaths[1], "/files/report.pdf")
	}

	if buf.String() != "file-content-here" {
		t.Errorf("downloaded content = %q, want %q", buf.String(), "file-content-here")
	}
}
