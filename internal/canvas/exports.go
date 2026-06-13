package canvas

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"
)

// EpubExport represents a Canvas ePub export job.
type EpubExport struct {
	ID          string      `json:"id"`
	CreatedAt   string      `json:"created_at"`
	WorkflowState string  `json:"workflow_state"`
	ProgressURL string      `json:"progress_url"`
	UserID      string      `json:"user_id"`
	Attachment  *Attachment `json:"attachment,omitempty"`
}

// CourseEpubExport represents a course with its latest ePub export.
type CourseEpubExport struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	EpubExport *EpubExport `json:"epub_export,omitempty"`
}

// ContentExport represents a Canvas content export job.
type ContentExport struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	ExportType    string      `json:"export_type"`
	WorkflowState string      `json:"workflow_state"`
	ProgressURL   string      `json:"progress_url"`
	UserID        string      `json:"user_id"`
	Attachment    *Attachment `json:"attachment,omitempty"`
}

// Progress represents a Canvas progress object.
type Progress struct {
	ID        string  `json:"id"`
	ContextID string  `json:"context_id"`
	ContextType string `json:"context_type"`
	UserID    string  `json:"user_id"`
	Tag       string  `json:"tag"`
	Completion *float64 `json:"completion,omitempty"`
	WorkflowState string `json:"workflow_state"`
}

// StartEpubExport creates a new ePub export for a course.
func StartEpubExport(ctx context.Context, client *Client, courseID string) (EpubExport, error) {
	var export EpubExport
	_, err := Request(ctx, client, RequestOptions{
		Method:     "POST",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/epub_exports", courseID),
		DecodeInto: &export,
	})
	if err != nil {
		return export, fmt.Errorf("start epub export for course %s: %w", courseID, err)
	}
	return export, nil
}

// GetEpubExport returns the status of an ePub export.
func GetEpubExport(ctx context.Context, client *Client, courseID, exportID string) (EpubExport, error) {
	var export EpubExport
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/epub_exports/%s", courseID, exportID),
		DecodeInto: &export,
	})
	if err != nil {
		return export, fmt.Errorf("get epub export %s: %w", exportID, err)
	}
	return export, nil
}

// ListEpubExports returns all courses with their latest ePub export status.
func ListEpubExports(ctx context.Context, client *Client) ([]CourseEpubExport, PaginationMeta, error) {
	var exports []CourseEpubExport
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/epub_exports",
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &exports,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list epub exports: %w", err)
	}
	return exports, meta.Pagination, nil
}

// StartContentExport creates a new content export for a course.
func StartContentExport(ctx context.Context, client *Client, courseID, exportType string) (ContentExport, error) {
	var export ContentExport
	body := url.Values{}
	body.Set("export_type", exportType)
	_, err := Request(ctx, client, RequestOptions{
		Method:    "POST",
		PathOrURL: fmt.Sprintf("/api/v1/courses/%s/content_exports", courseID),
		Body:      formReader(body),
		Headers: map[string][]string{
			"Content-Type": {"application/x-www-form-urlencoded"},
		},
		DecodeInto: &export,
	})
	if err != nil {
		return export, fmt.Errorf("start content export for course %s: %w", courseID, err)
	}
	return export, nil
}

// GetContentExport returns the status of a content export.
func GetContentExport(ctx context.Context, client *Client, courseID, exportID string) (ContentExport, error) {
	var export ContentExport
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/content_exports/%s", courseID, exportID),
		DecodeInto: &export,
	})
	if err != nil {
		return export, fmt.Errorf("get content export %s: %w", exportID, err)
	}
	return export, nil
}

// ListContentExports returns all content exports for a course.
func ListContentExports(ctx context.Context, client *Client, courseID string) ([]ContentExport, PaginationMeta, error) {
	var exports []ContentExport
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/content_exports", courseID),
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &exports,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list content exports for course %s: %w", courseID, err)
	}
	return exports, meta.Pagination, nil
}

// GetProgress returns the status of a progress operation.
func GetProgress(ctx context.Context, client *Client, progressURL string) (Progress, error) {
	var progress Progress
	parsed, err := url.Parse(progressURL)
	if err != nil {
		return progress, fmt.Errorf("parse progress URL: %w", err)
	}
	_, err = Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  parsed.Path,
		DecodeInto: &progress,
	})
	if err != nil {
		return progress, fmt.Errorf("get progress: %w", err)
	}
	return progress, nil
}

// DownloadExport downloads an export file from an attachment URL.
func DownloadExport(ctx context.Context, client *Client, attachmentURL string, w io.Writer) error {
	parsed, err := url.Parse(attachmentURL)
	if err != nil {
		return fmt.Errorf("parse attachment URL: %w", err)
	}

	resp, err := client.Do(ctx, "GET", parsed.Path, parsed.Query(), nil)
	if err != nil {
		return fmt.Errorf("download export: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download export: status %d", resp.StatusCode)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("write export content: %w", err)
	}
	return nil
}

// WaitForExport polls a progress URL until completion or failure.
// It returns nil when the export is complete, or an error if it fails.
func WaitForExport(ctx context.Context, client *Client, progressURL string, pollInterval time.Duration) error {
	for {
		progress, err := GetProgress(ctx, client, progressURL)
		if err != nil {
			return err
		}

		switch progress.WorkflowState {
		case "completed":
			return nil
		case "failed":
			return fmt.Errorf("export failed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// continue polling
		}
	}
}

// formReader creates an io.Reader from URL-encoded form values.
func formReader(values url.Values) io.Reader {
	return urlEncodedReader(values.Encode())
}

// urlEncodedReader creates an io.Reader from a URL-encoded string.
func urlEncodedReader(s string) io.Reader {
	return &stringReader{s: s}
}

type stringReader struct {
	s string
	i int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}
