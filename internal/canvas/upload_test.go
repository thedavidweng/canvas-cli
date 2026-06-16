package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestUploadFile_RelativeUploadURL tests the 3-step upload flow when step 1
// returns a relative upload_url (path only, not a full URL).
func TestUploadFile_RelativeUploadURL(t *testing.T) {
	var step1Path string
	var step1Method string
	var step2Path string
	var step2Method string
	var step2ContentType string
	var step2HasFile bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/courses/42/files":
			// Step 1: initiate upload
			step1Path = r.URL.Path
			step1Method = r.Method
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(uploadInitResponse{
				UploadURL: "/uploads/123",
				UploadParams: map[string]string{
					"token": "abc123",
				},
			})

		case "/uploads/123":
			// Step 2: receive the file upload (multipart)
			step2Path = r.URL.Path
			step2Method = r.Method
			step2ContentType = r.Header.Get("Content-Type")

			// Parse multipart form to verify the file content was sent.
			if err := r.ParseMultipartForm(1 << 20); err == nil {
				if _, _, err := r.FormFile("file"); err == nil {
					step2HasFile = true
				}
			}

			// Return 201 with file JSON.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(File{
				ID:          "555",
				DisplayName: "test.txt",
				Filename:    "test.txt",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	fileID, err := UploadFile(context.Background(), c, "42", "/tmp/test.txt", []byte("hello world"))
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}

	if fileID != "555" {
		t.Errorf("fileID = %q, want %q", fileID, "555")
	}

	// Verify step 1
	if step1Method != "POST" {
		t.Errorf("step1 method = %q, want POST", step1Method)
	}
	if step1Path != "/api/v1/courses/42/files" {
		t.Errorf("step1 path = %q, want /api/v1/courses/42/files", step1Path)
	}

	// Verify step 2
	if step2Method != "POST" {
		t.Errorf("step2 method = %q, want POST", step2Method)
	}
	if step2Path != "/uploads/123" {
		t.Errorf("step2 path = %q, want /uploads/123", step2Path)
	}
	if !strings.HasPrefix(step2ContentType, "multipart/form-data") {
		t.Errorf("step2 Content-Type = %q, want multipart/form-data", step2ContentType)
	}
	if !step2HasFile {
		t.Error("step2 did not receive file in multipart form")
	}
}

// TestUploadFile_WithRedirect tests the upload flow when step 2 returns a
// redirect. The default HTTP client follows the redirect, and the redirect
// target returns 201 with file JSON.
func TestUploadFile_WithRedirect(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/courses/1/files":
			// Step 1
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(uploadInitResponse{
				UploadURL: "/uploads/456",
				UploadParams: map[string]string{
					"token": "def456",
				},
			})

		case "/uploads/456":
			// Step 2: return 302 redirect. The HTTP client follows this
			// automatically (standard behavior for 302), issuing a GET to
			// the redirect target.
			w.Header().Set("Location", srv.URL+"/api/v1/files/999")
			w.WriteHeader(http.StatusFound)

		case "/api/v1/files/999":
			// Redirect target: return 201 with file JSON.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(File{
				ID:          "999",
				DisplayName: "redirected.txt",
				Filename:    "redirected.txt",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	fileID, err := UploadFile(context.Background(), c, "1", "/tmp/redirected.txt", []byte("redirect content"))
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}

	if fileID != "999" {
		t.Errorf("fileID = %q, want %q", fileID, "999")
	}
}

// TestUploadFile_InitStep4xx tests that UploadFile returns an error when step 1
// returns a 4xx status.
func TestUploadFile_InitStep4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"errors": ["unauthorized"]}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := UploadFile(context.Background(), c, "1", "/tmp/test.txt", []byte("data"))
	if err == nil {
		t.Fatal("UploadFile() expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %q, want it to contain 403", err.Error())
	}
}

// TestUploadFile_MissingUploadURL tests that UploadFile returns an error when
// step 1 response JSON does not include an upload_url field.
func TestUploadFile_MissingUploadURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return valid JSON with an empty upload_url.
		fmt.Fprint(w, `{"upload_url": "", "upload_params": {}}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	_, err := UploadFile(context.Background(), c, "1", "/tmp/test.txt", []byte("data"))
	if err == nil {
		t.Fatal("UploadFile() expected error for missing upload_url, got nil")
	}
	if !strings.Contains(err.Error(), "missing upload_url") {
		t.Errorf("error = %q, want it to contain 'missing upload_url'", err.Error())
	}
}

// TestHandleUploadResponse_201 tests that handleUploadResponse extracts the file
// ID from a 201 response body containing valid File JSON.
func TestHandleUploadResponse_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(File{ID: "777"})
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}

	fileID, err := handleUploadResponse(context.Background(), resp, nil)
	if err != nil {
		t.Fatalf("handleUploadResponse() error: %v", err)
	}
	if fileID != "777" {
		t.Errorf("fileID = %q, want %q", fileID, "777")
	}
}

// TestHandleUploadResponse_Redirect tests that handleUploadResponse follows a
// 3xx redirect by calling the client's DoURL to fetch the final file JSON.
func TestHandleUploadResponse_Redirect(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/files/888" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(File{ID: "888"})
			return
		}
		// Return a redirect for any other path.
		w.Header().Set("Location", srv.URL+"/api/v1/files/888")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)

	// Get a 302 response from the server, disabling automatic redirect following
	// so we get the raw 302 response.
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := noRedirectClient.Get(srv.URL + "/some/redirect")
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}

	fileID, err := handleUploadResponse(context.Background(), resp, c)
	if err != nil {
		t.Fatalf("handleUploadResponse() error: %v", err)
	}
	if fileID != "888" {
		t.Errorf("fileID = %q, want %q", fileID, "888")
	}
}

// TestHandleUploadResponse_RedirectNoLocation tests that handleUploadResponse
// returns an error when a 3xx response has no Location header.
func TestHandleUploadResponse_RedirectNoLocation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 302 with no Location header.
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	// Use a client that doesn't follow redirects so we get the raw 302.
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := noRedirectClient.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}

	_, err = handleUploadResponse(context.Background(), resp, nil)
	if err == nil {
		t.Fatal("handleUploadResponse() expected error for redirect without Location, got nil")
	}
	if !strings.Contains(err.Error(), "missing Location header") {
		t.Errorf("error = %q, want it to contain 'missing Location header'", err.Error())
	}
}

// TestHandleUploadResponse_UnexpectedStatus tests that handleUploadResponse
// returns an error for an unexpected status code (e.g. 400).
func TestHandleUploadResponse_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}

	_, err = handleUploadResponse(context.Background(), resp, nil)
	if err == nil {
		t.Fatal("handleUploadResponse() expected error for 400 status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 400") {
		t.Errorf("error = %q, want it to contain 'unexpected status 400'", err.Error())
	}
}

// TestExtractFileID_ValidJSON tests that extractFileID returns the file ID from
// a response body containing valid File JSON.
func TestExtractFileID_ValidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(File{
			ID:          "321",
			DisplayName: "photo.png",
			Filename:    "photo.png",
		})
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}

	fileID, err := extractFileID(resp)
	if err != nil {
		t.Fatalf("extractFileID() error: %v", err)
	}
	if fileID != "321" {
		t.Errorf("fileID = %q, want %q", fileID, "321")
	}
}

// TestExtractFileID_MissingID tests that extractFileID returns an error when
// the JSON response does not contain an id field.
func TestExtractFileID_MissingID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"display_name": "no-id.txt"}`)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}

	_, err = extractFileID(resp)
	if err == nil {
		t.Fatal("extractFileID() expected error for missing id, got nil")
	}
	if !strings.Contains(err.Error(), "missing file ID") {
		t.Errorf("error = %q, want it to contain 'missing file ID'", err.Error())
	}
}

// TestExtractFileID_InvalidJSON tests that extractFileID returns an error when
// the response body is not valid JSON.
func TestExtractFileID_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "not json at all")
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}

	_, err = extractFileID(resp)
	if err == nil {
		t.Fatal("extractFileID() expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "decode upload response") {
		t.Errorf("error = %q, want it to contain 'decode upload response'", err.Error())
	}
}

// TestExtractFileIDFromLocation tests all variants of Location header URLs that
// the extractFileIDFromLocation function must parse.
func TestExtractFileIDFromLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantID   string
		wantErr  bool
	}{
		{
			name:     "relative path",
			location: "/api/v1/files/12345",
			wantID:   "12345",
		},
		{
			name:     "full URL with query params",
			location: "https://canvas.school.edu/api/v1/files/67890?token=abc",
			wantID:   "67890",
		},
		{
			name:     "full URL without query params",
			location: "https://canvas.school.edu/api/v1/files/99999",
			wantID:   "99999",
		},
		{
			name:     "unrelated path returns error",
			location: "/some/other/path",
			wantErr:  true,
		},
		{
			name:     "empty file ID returns error",
			location: "/api/v1/files/",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := extractFileIDFromLocation(tt.location)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractFileIDFromLocation(%q) expected error, got nil", tt.location)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractFileIDFromLocation(%q) error: %v", tt.location, err)
			}
			if gotID != tt.wantID {
				t.Errorf("extractFileIDFromLocation(%q) = %q, want %q", tt.location, gotID, tt.wantID)
			}
		})
	}
}
