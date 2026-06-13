package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewFilesCmd returns the `files` parent command.
func NewFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Manage files",
		Long:  `List, download, and upload Canvas files.`,
	}

	cmd.AddCommand(newFilesListCmd())
	cmd.AddCommand(newFilesGetCmd())
	cmd.AddCommand(newFilesDownloadCmd())
	cmd.AddCommand(newFilesDownloadCourseCmd())
	cmd.AddCommand(newFilesUploadCmd())

	return cmd
}

func newFilesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List files in a course",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			files, _, err := canvas.ListFiles(cmd.Context(), client, courseID, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "files.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(files, "files.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, f := range files {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", f.ID, f.DisplayName, f.Size, f.ContentType)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newFilesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get FILE_ID",
		Short: "Get a file by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			fileID := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			file, err := canvas.GetFile(cmd.Context(), client, fileID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "files.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(file, "files.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:           %s\n", file.ID)
			fmt.Fprintf(w, "Display Name: %s\n", file.DisplayName)
			fmt.Fprintf(w, "Filename:     %s\n", file.Filename)
			fmt.Fprintf(w, "Content Type: %s\n", file.ContentType)
			fmt.Fprintf(w, "Size:         %d\n", file.Size)
			fmt.Fprintf(w, "Created At:   %s\n", file.CreatedAt)
			fmt.Fprintf(w, "Updated At:   %s\n", file.UpdatedAt)
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newFilesDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download FILE_ID",
		Short: "Download a file to a local path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			fileID := args[0]
			outPath, _ := cmd.Flags().GetString("out")
			noOverwrite, _ := cmd.Flags().GetBool("no-overwrite")

			if outPath == "" {
				return fmt.Errorf("--out is required")
			}

			// Check --no-overwrite
			if noOverwrite {
				if _, err := os.Stat(outPath); err == nil {
					return fmt.Errorf("file already exists: %s", outPath)
				}
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer f.Close()

			if err := canvas.DownloadFile(cmd.Context(), client, fileID, f); err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Downloaded file %s to %s\n", fileID, outPath)
			return nil
		},
	}
	cmd.Flags().String("out", "", "output file path (required)")
	cmd.Flags().Bool("no-overwrite", false, "fail if output file already exists")
	return cmd
}

// FileDownloadResult holds the outcome of a course files download operation.
type FileDownloadResult struct {
	Total        int    `json:"total"`
	Downloaded   int    `json:"downloaded"`
	Failed       int    `json:"failed"`
	ManifestPath string `json:"manifest_path"`
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

func newFilesDownloadCourseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download-course",
		Short: "Download all files for a course",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			outDir, _ := cmd.Flags().GetString("out")
			if outDir == "" {
				return fmt.Errorf("--out is required")
			}
			noOverwrite, _ := cmd.Flags().GetBool("no-overwrite")
			jsonMode, _ := cmd.Flags().GetBool("json")

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			// List all files for the course
			files, _, err := canvas.ListFiles(cmd.Context(), client, courseID, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "files.download-course")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			result := &FileDownloadResult{Total: len(files)}
			var entries []FileManifestEntry
			var failures []error

			for _, f := range files {
				entry := FileManifestEntry{
					FileID:      f.ID,
					Filename:    f.Filename,
					DisplayName: f.DisplayName,
					ContentType: f.ContentType,
					Size:        f.Size,
				}

				localPath := filepath.Join(outDir, f.Filename)
				entry.LocalPath = localPath

				// Check --no-overwrite
				if noOverwrite {
					if _, statErr := os.Stat(localPath); statErr == nil {
						entry.DownloadStatus = "skipped"
						entries = append(entries, entry)
						result.Downloaded++
						continue
					}
				}

				// Download file
				outFile, createErr := os.Create(localPath)
				if createErr != nil {
					entry.DownloadStatus = "error"
					entry.Error = createErr.Error()
					entries = append(entries, entry)
					failures = append(failures, createErr)
					result.Failed++
					continue
				}

				dlErr := canvas.DownloadFile(cmd.Context(), client, f.ID, outFile)
				outFile.Close()
				if dlErr != nil {
					entry.DownloadStatus = "error"
					entry.Error = dlErr.Error()
					entries = append(entries, entry)
					failures = append(failures, dlErr)
					result.Failed++
					continue
				}

				entry.DownloadStatus = "ok"
				entries = append(entries, entry)
				result.Downloaded++
			}

			// Write manifest
			if mkErr := os.MkdirAll(outDir, 0755); mkErr != nil {
				return fmt.Errorf("create output directory: %w", mkErr)
			}

			manifestJSONPath := filepath.Join(outDir, "manifest.json")
			jsonData, jsonErr := json.MarshalIndent(entries, "", "  ")
			if jsonErr != nil {
				return fmt.Errorf("marshal manifest: %w", jsonErr)
			}
			if writeErr := os.WriteFile(manifestJSONPath, jsonData, 0644); writeErr != nil {
				return fmt.Errorf("write manifest.json: %w", writeErr)
			}
			result.ManifestPath = manifestJSONPath

			// manifest.ndjson
			manifestNDJSONPath := filepath.Join(outDir, "manifest.ndjson")
			ndjsonFile, ndErr := os.Create(manifestNDJSONPath)
			if ndErr != nil {
				return fmt.Errorf("create manifest.ndjson: %w", ndErr)
			}
			defer ndjsonFile.Close()
			enc := json.NewEncoder(ndjsonFile)
			for _, entry := range entries {
				if encErr := enc.Encode(entry); encErr != nil {
					return fmt.Errorf("encode manifest.ndjson entry: %w", encErr)
				}
			}

			// Output
			if jsonMode {
				meta := canvas.Meta{Profile: cfg.Profile, BaseURL: cfg.BaseURL}
				env := output.NewSuccess(result, "files.download-course", meta)
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Downloaded %d/%d files\n", result.Downloaded, result.Total)
			if result.Failed > 0 {
				fmt.Fprintf(w, "%d failures (see manifest for details)\n", result.Failed)
			}
			if result.ManifestPath != "" {
				fmt.Fprintf(w, "Manifest: %s\n", result.ManifestPath)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("out", "", "output directory (required)")
	cmd.Flags().Bool("no-overwrite", false, "skip files that already exist")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newFilesUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a file to a course",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			filePath, _ := cmd.Flags().GetString("file")
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}
			folder, _ := cmd.Flags().GetString("folder")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")
			jsonMode, _ := cmd.Flags().GetBool("json")

			// Safety check
			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			// Build preview
			path := fmt.Sprintf("/api/v1/courses/%s/files", courseID)
			preview := safety.FormatPreview(safety.Preview{
				Method:         "POST",
				Path:           path,
				ResourceIDs:    []string{courseID},
				PayloadSummary: fmt.Sprintf("file=%s folder=%s", filePath, folder),
			})

			// Dry-run: show preview and exit
			if dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			// Read file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read file %s: %w", filePath, err)
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			// Upload via 3-step flow
			fileID, err := canvas.UploadFile(cmd.Context(), client, courseID, filePath, content)
			if err != nil {
				return fmt.Errorf("upload file: %w", err)
			}

			// Write audit
			writeAudit(cfg, "files.upload", "POST", path,
				fmt.Sprintf(`{"file":"%s","folder":"%s"}`, filepath.Base(filePath), folder), false)

			// Output
			if jsonMode {
				result := map[string]string{
					"id":   fileID,
					"name": filepath.Base(filePath),
				}
				env := output.NewSuccess(result, "files.upload", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Uploaded file %s (ID: %s)\n", filepath.Base(filePath), fileID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("file", "", "file path to upload (required)")
	cmd.Flags().String("folder", "", "target folder path (optional)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
