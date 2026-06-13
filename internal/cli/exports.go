package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// newCoursesExportCmd returns `courses export`.
func newCoursesExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export course content",
		Long: `Export course content in various formats.

Supported formats:
  epub             - ePub format (for e-readers)
  common_cartridge - Common Cartridge (.imscc) format
  qti              - QTI format (quizzes only)
  zip              - Zip archive of files

The export is asynchronous — the CLI polls until complete, then downloads
the file to the current directory (or --out path).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			format, _ := cmd.Flags().GetString("format")
			outPath, _ := cmd.Flags().GetString("out")
			noWait, _ := cmd.Flags().GetBool("no-wait")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if format == "" {
				return fmt.Errorf("--format is required (epub, common_cartridge, qti, zip)")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			w := cmd.OutOrStdout()

			if format == "epub" {
				return exportEpub(cmd.Context(), client, w, courseID, outPath, noWait, jsonMode, cfg)
			}
			return exportContent(cmd.Context(), client, w, courseID, format, outPath, noWait, jsonMode, cfg)
		},
	}

	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("format", "", "export format: epub, common_cartridge, qti, zip")
	cmd.Flags().String("out", "", "output file path (default: auto-generated)")
	cmd.Flags().Bool("no-wait", false, "start export but don't wait for completion")
	cmd.Flags().Bool("json", false, "output JSON envelope")

	return cmd
}

// newCoursesExportsCmd returns `courses exports`.
func newCoursesExportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exports",
		Short: "List past content exports",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			exports, _, err := canvas.ListContentExports(cmd.Context(), client, courseID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "courses.exports")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(exports, "courses.exports", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Type", "State", "Created"},
			}
			for _, e := range exports {
				tbl.Rows = append(tbl.Rows, []string{e.ID, e.ExportType, e.WorkflowState, e.CreatedAt})
			}
			return tbl.Render(w, false)
		},
	}

	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")

	return cmd
}

func exportEpub(ctx context.Context, client *canvas.Client, w io.Writer, courseID, outPath string, noWait, jsonMode bool, cfg *config.ResolvedConfig) error {
	fmt.Fprintf(w, "Starting ePub export for course %s...\n", courseID)

	export, err := canvas.StartEpubExport(ctx, client, courseID)
	if err != nil {
		return err
	}

	if noWait {
		if jsonMode {
			env := output.NewSuccess(export, "courses.export")
			return output.WriteJSON(w, env, false)
		}
		fmt.Fprintf(w, "Export started (ID: %s)\n", export.ID)
		fmt.Fprintf(w, "Poll progress at: %s\n", export.ProgressURL)
		return nil
	}

	// Wait for completion
	fmt.Fprintf(w, "Waiting for export to complete")
	if err := waitForComplete(ctx, client, export.ProgressURL, w); err != nil {
		return err
	}

	// Re-fetch to get attachment
	export, err = canvas.GetEpubExport(ctx, client, courseID, export.ID)
	if err != nil {
		return err
	}

	if export.Attachment == nil || export.Attachment.URL == "" {
		return fmt.Errorf("export completed but no download URL available")
	}

	// Download
	if outPath == "" {
		outPath = fmt.Sprintf("course-%s.epub", courseID)
	}

	fmt.Fprintf(w, "\nDownloading to %s...\n", outPath)
	if err := downloadToFile(ctx, client, export.Attachment.URL, outPath); err != nil {
		return err
	}

	fmt.Fprintf(w, "Done! Saved to %s\n", outPath)
	return nil
}

func exportContent(ctx context.Context, client *canvas.Client, w io.Writer, courseID, format, outPath string, noWait, jsonMode bool, cfg *config.ResolvedConfig) error {
	fmt.Fprintf(w, "Starting %s export for course %s...\n", format, courseID)

	export, err := canvas.StartContentExport(ctx, client, courseID, format)
	if err != nil {
		return err
	}

	if noWait {
		if jsonMode {
			env := output.NewSuccess(export, "courses.export")
			return output.WriteJSON(w, env, false)
		}
		fmt.Fprintf(w, "Export started (ID: %s, type: %s)\n", export.ID, export.ExportType)
		fmt.Fprintf(w, "Poll progress at: %s\n", export.ProgressURL)
		return nil
	}

	// Wait for completion
	fmt.Fprintf(w, "Waiting for export to complete")
	if err := waitForComplete(ctx, client, export.ProgressURL, w); err != nil {
		return err
	}

	// Re-fetch to get attachment
	export, err = canvas.GetContentExport(ctx, client, courseID, export.ID)
	if err != nil {
		return err
	}

	if export.Attachment == nil || export.Attachment.URL == "" {
		return fmt.Errorf("export completed but no download URL available")
	}

	// Determine file extension
	ext := ".zip"
	switch format {
	case "common_cartridge":
		ext = ".imscc"
	case "qti":
		ext = ".zip"
	case "zip":
		ext = ".zip"
	}

	if outPath == "" {
		outPath = fmt.Sprintf("course-%s%s", courseID, ext)
	}

	fmt.Fprintf(w, "\nDownloading to %s...\n", outPath)
	if err := downloadToFile(ctx, client, export.Attachment.URL, outPath); err != nil {
		return err
	}

	fmt.Fprintf(w, "Done! Saved to %s\n", outPath)
	return nil
}

func waitForComplete(ctx context.Context, client *canvas.Client, progressURL string, w io.Writer) error {
	for {
		progress, err := canvas.GetProgress(ctx, client, progressURL)
		if err != nil {
			return fmt.Errorf("check progress: %w", err)
		}

		switch progress.WorkflowState {
		case "completed":
			fmt.Fprintf(w, ". done!\n")
			return nil
		case "failed":
			fmt.Fprintf(w, "\n")
			return fmt.Errorf("export failed")
		}

		fmt.Fprintf(w, ".")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func downloadToFile(ctx context.Context, client *canvas.Client, url, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	return canvas.DownloadExport(ctx, client, url, f)
}
