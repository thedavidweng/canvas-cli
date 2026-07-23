package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DownloadCourseFilesOptions controls a bulk course-files download.
type DownloadCourseFilesOptions struct {
	CourseID    string
	OutDir      string
	NoOverwrite bool
}

// FileManifestEntry represents a single file entry in the download manifest.
type FileManifestEntry struct {
	FileID         string `json:"file_id"`
	Filename       string `json:"filename"`
	DisplayName    string `json:"display_name"`
	ContentType    string `json:"content_type"`
	Size           int64  `json:"size"`
	LocalPath      string `json:"local_path"`
	DownloadStatus string `json:"download_status"`
	Error          string `json:"error,omitempty"`
}

// FileDownloadResult holds the outcome of a course files download operation.
type FileDownloadResult struct {
	Total        int                 `json:"total"`
	Downloaded   int                 `json:"downloaded"`
	Failed       int                 `json:"failed"`
	Entries      []FileManifestEntry `json:"-"`
	ManifestPath string              `json:"manifest_path"`
}

// DownloadCourseFiles lists all files for a course, downloads each one into
// outDir, and writes manifest.json + manifest.ndjson alongside them. Files
// that already exist are skipped when noOverwrite is true. The returned
// FileDownloadResult summarises the operation; its Entries slice holds
// per-file status for callers that need the detail.
func DownloadCourseFiles(ctx context.Context, client *Client, opts DownloadCourseFilesOptions) (*FileDownloadResult, error) {
	files, _, err := ListFiles(ctx, client, opts.CourseID, nil)
	if err != nil {
		return nil, fmt.Errorf("list files for course %s: %w", opts.CourseID, err)
	}

	result := &FileDownloadResult{Total: len(files), Entries: make([]FileManifestEntry, 0, len(files))}

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	for _, f := range files {
		entry := FileManifestEntry{
			FileID:      f.ID,
			Filename:    f.Filename,
			DisplayName: f.DisplayName,
			ContentType: f.ContentType,
			Size:        f.Size,
		}

		localPath := filepath.Join(opts.OutDir, f.Filename)
		entry.LocalPath = localPath

		if opts.NoOverwrite {
			if _, statErr := os.Stat(localPath); statErr == nil {
				entry.DownloadStatus = "skipped"
				result.Entries = append(result.Entries, entry)
				result.Downloaded++
				continue
			}
		}

		outFile, createErr := os.Create(localPath)
		if createErr != nil {
			entry.DownloadStatus = "error"
			entry.Error = createErr.Error()
			result.Entries = append(result.Entries, entry)
			result.Failed++
			continue
		}

		dlErr := DownloadFile(ctx, client, f.ID, outFile)
		outFile.Close()
		if dlErr != nil {
			entry.DownloadStatus = "error"
			entry.Error = dlErr.Error()
			result.Entries = append(result.Entries, entry)
			result.Failed++
			continue
		}

		entry.DownloadStatus = "ok"
		result.Entries = append(result.Entries, entry)
		result.Downloaded++
	}

	// Write manifest.json
	manifestJSONPath := filepath.Join(opts.OutDir, "manifest.json")
	jsonData, jsonErr := json.MarshalIndent(result.Entries, "", "  ")
	if jsonErr != nil {
		return result, fmt.Errorf("marshal manifest: %w", jsonErr)
	}
	if writeErr := os.WriteFile(manifestJSONPath, jsonData, 0o644); writeErr != nil {
		return result, fmt.Errorf("write manifest.json: %w", writeErr)
	}
	result.ManifestPath = manifestJSONPath

	// Write manifest.ndjson
	manifestNDJSONPath := filepath.Join(opts.OutDir, "manifest.ndjson")
	ndjsonFile, ndErr := os.Create(manifestNDJSONPath)
	if ndErr != nil {
		return result, fmt.Errorf("create manifest.ndjson: %w", ndErr)
	}
	defer ndjsonFile.Close()
	enc := json.NewEncoder(ndjsonFile)
	for _, entry := range result.Entries {
		if encErr := enc.Encode(entry); encErr != nil {
			return result, fmt.Errorf("encode manifest.ndjson entry: %w", encErr)
		}
	}

	return result, nil
}
