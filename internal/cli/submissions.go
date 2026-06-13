package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// DownloadResult holds the outcome of a submissions download operation.
type DownloadResult struct {
	Total        int    `json:"total"`
	Downloaded   int    `json:"downloaded"`
	Failed       int    `json:"failed"`
	ManifestPath string `json:"manifest_path"`
}

// DownloadOptions controls submission download behavior.
type DownloadOptions struct {
	NoOverwrite bool
}

// PartialFailureError is returned when some file downloads fail but others succeed.
type PartialFailureError struct {
	Result *DownloadResult
	Errors []error
}

func (e *PartialFailureError) Error() string {
	return fmt.Sprintf("%d of %d downloads failed", e.Result.Failed, e.Result.Total)
}

// ExitCode returns 8, matching the partial-failure exit code contract.
func (e *PartialFailureError) ExitCode() int {
	return output.CodePartialFailure
}

// ManifestEntry represents a single file entry in the download manifest.
type ManifestEntry struct {
	SubmissionID   string `json:"submission_id"`
	UserID         string `json:"user_id"`
	UserName       string `json:"user_name"`
	SortableName   string `json:"sortable_name"`
	AttachmentID   string `json:"attachment_id"`
	Filename       string `json:"filename"`
	OriginalURL    string `json:"original_url"`
	LocalPath      string `json:"local_path"`
	Size           int64  `json:"size"`
	DownloadStatus string `json:"download_status"`
	Error          string `json:"error,omitempty"`
}

// NewSubmissionsCmd returns the `submissions` parent command.
func NewSubmissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submissions",
		Short: "Manage submissions",
		Long:  `List, get, download, and comment on Canvas submissions.`,
	}

	cmd.AddCommand(newSubmissionsListCmd())
	cmd.AddCommand(newSubmissionsGetCmd())
	cmd.AddCommand(newSubmissionsDownloadCmd())
	cmd.AddCommand(newSubmissionsCommentCmd())

	return cmd
}

func newSubmissionsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List submissions for an assignment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			submissions, _, err := canvas.ListSubmissions(cmd.Context(), client, courseID, assignmentID, canvas.RequestOptions{})
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "submissions.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(submissions, "submissions.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, sub := range submissions {
				userName := ""
				if sub.User != nil {
					userName = sub.User.Name
				}
				scoreStr := ""
				if sub.Score != nil {
					scoreStr = fmt.Sprintf("%g", *sub.Score)
				}
				submittedStr := ""
				if sub.SubmittedAt != nil {
					submittedStr = *sub.SubmittedAt
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", sub.ID, userName, sub.WorkflowState, scoreStr, submittedStr)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newSubmissionsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a submission for a specific user and assignment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			userID, _ := cmd.Flags().GetString("user")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}
			if userID == "" {
				return fmt.Errorf("--user is required")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			sub, err := canvas.GetSubmission(cmd.Context(), client, courseID, assignmentID, userID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "submissions.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(sub, "submissions.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:             %s\n", sub.ID)
			fmt.Fprintf(w, "User ID:        %s\n", sub.UserID)
			fmt.Fprintf(w, "Assignment ID:  %s\n", sub.AssignmentID)
			fmt.Fprintf(w, "State:          %s\n", sub.WorkflowState)
			if sub.SubmittedAt != nil {
				fmt.Fprintf(w, "Submitted At:   %s\n", *sub.SubmittedAt)
			}
			if sub.Score != nil {
				fmt.Fprintf(w, "Score:          %g\n", *sub.Score)
			}
			if sub.Grade != nil {
				fmt.Fprintf(w, "Grade:          %s\n", *sub.Grade)
			}
			fmt.Fprintf(w, "Late:           %t\n", sub.Late)
			fmt.Fprintf(w, "Missing:        %t\n", sub.Missing)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().String("user", "", "user ID or 'self' (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newSubmissionsDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download submission files for an assignment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			outDir, _ := cmd.Flags().GetString("out")
			noOverwrite, _ := cmd.Flags().GetBool("no-overwrite")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}
			if outDir == "" {
				return fmt.Errorf("--out is required")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			result, err := DownloadSubmissions(cmd.Context(), client, courseID, assignmentID, outDir, DownloadOptions{
				NoOverwrite: noOverwrite,
			})

			if jsonMode {
				meta := canvas.Meta{Profile: cfg.Profile, BaseURL: cfg.BaseURL}
				if pfErr, ok := err.(*PartialFailureError); ok {
					meta.Warnings = []string{pfErr.Error()}
					env := output.NewSuccess(result, "submissions.download", meta)
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				if err != nil {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "submissions.download")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				env := output.NewSuccess(result, "submissions.download", meta)
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			if result != nil {
				fmt.Fprintf(w, "Downloaded %d/%d files\n", result.Downloaded, result.Total)
				if result.Failed > 0 {
					fmt.Fprintf(w, "%d failures (see manifest for details)\n", result.Failed)
				}
				if result.ManifestPath != "" {
					fmt.Fprintf(w, "Manifest: %s\n", result.ManifestPath)
				}
			}
			return err
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().String("out", "", "output directory (required)")
	cmd.Flags().Bool("no-overwrite", false, "skip files that already exist")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newSubmissionsCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Add a comment to a submission",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			userID, _ := cmd.Flags().GetString("user")
			comment, _ := cmd.Flags().GetString("comment")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}
			if userID == "" {
				return fmt.Errorf("--user is required")
			}
			if comment == "" {
				return fmt.Errorf("--comment is required")
			}

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions/%s", courseID, assignmentID, userID)

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:         "PUT",
					Path:           path,
					ResourceIDs:    []string{courseID, assignmentID, userID},
					PayloadSummary: fmt.Sprintf(`{"comment":{"text_comment":"%s"}}`, comment),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			sub, err := canvas.AddComment(cmd.Context(), client, courseID, assignmentID, userID, comment)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "submissions.comment")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			writeAudit(cfg, "submissions.comment", "PUT", path,
				fmt.Sprintf(`{"comment":{"text_comment":"%s"}}`, comment), false)

			if jsonMode {
				env := output.NewSuccess(sub, "submissions.comment", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Comment added for user %s on assignment %s\n", userID, assignmentID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().String("user", "", "user ID (required)")
	cmd.Flags().String("comment", "", "comment text (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// DownloadSubmissions downloads all submission attachments for an assignment.
// It lists submissions, downloads each attachment into a deterministic directory
// structure, and writes manifest.json and manifest.ndjson.
func DownloadSubmissions(ctx context.Context, client *canvas.Client, courseID, assignmentID, outDir string, opts DownloadOptions) (*DownloadResult, error) {
	submissions, _, err := canvas.ListSubmissions(ctx, client, courseID, assignmentID, canvas.RequestOptions{})
	if err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}

	result := &DownloadResult{}
	var entries []ManifestEntry
	var failures []error

	for _, sub := range submissions {
		sortableName := "unknown"
		userName := ""
		if sub.User != nil {
			if sub.User.SortableName != "" {
				sortableName = sub.User.SortableName
			}
			userName = sub.User.Name
		}

		for _, att := range sub.Attachments {
			result.Total++

			entry := ManifestEntry{
				SubmissionID: sub.ID,
				UserID:       sub.UserID,
				UserName:     userName,
				SortableName: sortableName,
				AttachmentID: att.ID,
				Filename:     att.Filename,
				OriginalURL:  att.URL,
				Size:         att.Size,
			}

			// Build deterministic path: <assignment-id>/<sortable-name>_<user-id>/<submission-id>_<filename>
			dirName := sortableName + "_" + sub.UserID
			localDir := filepath.Join(outDir, assignmentID, dirName)
			localFilename := sub.ID + "_" + att.Filename
			localPath := filepath.Join(localDir, localFilename)
			entry.LocalPath = localPath

			// Check --no-overwrite
			if opts.NoOverwrite {
				if _, statErr := os.Stat(localPath); statErr == nil {
					entry.DownloadStatus = "skipped"
					entries = append(entries, entry)
					result.Downloaded++
					continue
				}
			}

			// Create directory
			if mkErr := os.MkdirAll(localDir, 0755); mkErr != nil {
				entry.DownloadStatus = "error"
				entry.Error = mkErr.Error()
				entries = append(entries, entry)
				failures = append(failures, mkErr)
				result.Failed++
				continue
			}

			// Download file
			if dlErr := downloadAttachment(ctx, client, att, localPath); dlErr != nil {
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
	}

	// Write manifests into the assignment directory
	assignDir := filepath.Join(outDir, assignmentID)
	if mkErr := os.MkdirAll(assignDir, 0755); mkErr != nil {
		return result, fmt.Errorf("create assignment directory: %w", mkErr)
	}

	// manifest.json
	manifestJSONPath := filepath.Join(assignDir, "manifest.json")
	jsonData, jsonErr := json.MarshalIndent(entries, "", "  ")
	if jsonErr != nil {
		return result, fmt.Errorf("marshal manifest: %w", jsonErr)
	}
	if writeErr := os.WriteFile(manifestJSONPath, jsonData, 0644); writeErr != nil {
		return result, fmt.Errorf("write manifest.json: %w", writeErr)
	}
	result.ManifestPath = manifestJSONPath

	// manifest.ndjson
	manifestNDJSONPath := filepath.Join(assignDir, "manifest.ndjson")
	ndjsonFile, ndErr := os.Create(manifestNDJSONPath)
	if ndErr != nil {
		return result, fmt.Errorf("create manifest.ndjson: %w", ndErr)
	}
	defer ndjsonFile.Close()
	enc := json.NewEncoder(ndjsonFile)
	for _, entry := range entries {
		if encErr := enc.Encode(entry); encErr != nil {
			return result, fmt.Errorf("encode manifest.ndjson entry: %w", encErr)
		}
	}

	if result.Failed > 0 {
		return result, &PartialFailureError{Result: result, Errors: failures}
	}

	return result, nil
}

// downloadAttachment downloads a single attachment to a local file path.
func downloadAttachment(ctx context.Context, client *canvas.Client, att canvas.Attachment, localPath string) error {
	parsed, err := url.Parse(att.URL)
	if err != nil {
		return fmt.Errorf("parse attachment URL: %w", err)
	}

	query := parsed.Query()
	if len(query) == 0 {
		query = nil
	}

	resp, err := client.Do(ctx, "GET", parsed.Path, query, nil)
	if err != nil {
		return fmt.Errorf("download attachment %s: %w", att.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download attachment %s: status %d", att.ID, resp.StatusCode)
	}

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
