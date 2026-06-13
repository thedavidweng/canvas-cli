package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// GetFile returns metadata for a single file by ID.
func GetFile(ctx context.Context, client *Client, fileID string) (File, error) {
	var file File
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/files/%s", fileID),
		DecodeInto: &file,
	})
	if err != nil {
		return file, fmt.Errorf("get file %s: %w", fileID, err)
	}
	return file, nil
}

// ListFiles returns all files for a course.
// It sends GET /api/v1/courses/{courseID}/files with the given query parameters.
// The per_page parameter defaults to 100.
func ListFiles(ctx context.Context, client *Client, courseID string, query url.Values) ([]File, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	var files []File
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/files", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &files,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list files for course %s: %w", courseID, err)
	}

	return files, meta.Pagination, nil
}

// DownloadFile downloads a file by its ID into the provided writer.
// It first fetches the file metadata from GET /api/v1/files/{fileID} to obtain
// the download URL, then streams the file content into w.
func DownloadFile(ctx context.Context, client *Client, fileID string, w io.Writer) error {
	// Step 1: Get file metadata to obtain the download URL
	resp, err := client.Do(ctx, "GET", fmt.Sprintf("/api/v1/files/%s", fileID), nil, nil)
	if err != nil {
		return fmt.Errorf("get file metadata %s: %w", fileID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("get file metadata %s: status %d", fileID, resp.StatusCode)
	}

	var file File
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return fmt.Errorf("decode file metadata %s: %w", fileID, err)
	}

	if file.URL == "" {
		return fmt.Errorf("file %s: download URL is empty", fileID)
	}

	// Step 2: Download the file content from the URL.
	// Canvas returns a full URL; extract path so client.Do does not double-prefix.
	parsed, err := url.Parse(file.URL)
	if err != nil {
		return fmt.Errorf("parse file URL %s: %w", fileID, err)
	}

	dlResp, err := client.Do(ctx, "GET", parsed.Path, parsed.Query(), nil)
	if err != nil {
		return fmt.Errorf("download file %s: %w", fileID, err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode >= 400 {
		return fmt.Errorf("download file %s: status %d", fileID, dlResp.StatusCode)
	}

	if _, err := io.Copy(w, dlResp.Body); err != nil {
		return fmt.Errorf("write file content %s: %w", fileID, err)
	}

	return nil
}
