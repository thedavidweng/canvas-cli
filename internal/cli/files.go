package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")

			files, _, err := canvas.ListFiles(cmd.Context(), client, courseID, nil)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "files.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, files, "files.list", jsonMode, func(w io.Writer) error {
				for i := range files {
					f := &files[i]
					fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", f.ID, f.DisplayName, f.Size, f.ContentType)
				}
				return nil
			})
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			fileID := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")

			file, err := canvas.GetFile(cmd.Context(), client, fileID)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "files.get", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, file, "files.get", jsonMode, func(w io.Writer) error {
				fmt.Fprintf(w, "ID:           %s\n", file.ID)
				fmt.Fprintf(w, "Display Name: %s\n", file.DisplayName)
				fmt.Fprintf(w, "Filename:     %s\n", file.Filename)
				fmt.Fprintf(w, "Content Type: %s\n", file.ContentType)
				fmt.Fprintf(w, "Size:         %d\n", file.Size)
				fmt.Fprintf(w, "Created At:   %s\n", file.CreatedAt)
				fmt.Fprintf(w, "Updated At:   %s\n", file.UpdatedAt)
				return nil
			})
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

			if noOverwrite {
				if _, err := os.Stat(outPath); err == nil {
					return fmt.Errorf("file already exists: %s", outPath)
				}
			}

			client := newClientFromCfg(cfg)

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

func newFilesDownloadCourseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download-course",
		Short: "Download all files for a course",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

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

			result, err := canvas.DownloadCourseFiles(cmd.Context(), client, canvas.DownloadCourseFilesOptions{
				CourseID:    courseID,
				OutDir:      outDir,
				NoOverwrite: noOverwrite,
			})
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "files.download-course", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, result, "files.download-course", jsonMode, func(w io.Writer) error {
				fmt.Fprintf(w, "Downloaded %d/%d files\n", result.Downloaded, result.Total)
				if result.Failed > 0 {
					fmt.Fprintf(w, "%d failures (see manifest for details)\n", result.Failed)
				}
				if result.ManifestPath != "" {
					fmt.Fprintf(w, "Manifest: %s\n", result.ManifestPath)
				}
				return nil
			})
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

			path := fmt.Sprintf("/api/v1/courses/%s/files", courseID)

			spec := MutationSpec{
				Command:        "files.upload",
				Level:          safety.LowRiskWrite,
				Method:         "POST",
				Path:           path,
				DryRun:         dryRun,
				Confirm:        confirm,
				ResourceIDs:    []string{courseID},
				PayloadSummary: fmt.Sprintf("file=%s folder=%s", filePath, folder),
				AuditBody:      fmt.Sprintf(`{"file":%q,"folder":%q}`, filepath.Base(filePath), folder),
			}

			dryRunShortCircuit, err := CheckAndPreview(cfg, cmd.OutOrStdout(), &spec)
			if err != nil {
				return err
			}
			if dryRunShortCircuit {
				return nil
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read file %s: %w", filePath, err)
			}

			client := newClientFromCfg(cfg)
			fileID, err := canvas.UploadFile(cmd.Context(), client, courseID, filePath, content)
			if err != nil {
				RecordAudit(cfg, &spec, 0, false)
				return writeError(cmd.OutOrStdout(), cfg, fmt.Errorf("upload file: %w", err), "files.upload", jsonMode)
			}

			RecordAudit(cfg, &spec, 200, true)

			result := map[string]string{
				"id":   fileID,
				"name": filepath.Base(filePath),
			}
			return writeOutput(cmd.OutOrStdout(), cfg, result, "files.upload", jsonMode, func(w io.Writer) error {
				fmt.Fprintf(w, "Uploaded file %s (ID: %s)\n", filepath.Base(filePath), fileID)
				return nil
			})
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
