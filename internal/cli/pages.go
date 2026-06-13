package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewPagesCmd returns the `pages` parent command.
func NewPagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pages",
		Short: "Manage wiki pages",
		Long:  `List, get, and update Canvas wiki pages.`,
	}

	cmd.AddCommand(newPagesListCmd())
	cmd.AddCommand(newPagesGetCmd())
	cmd.AddCommand(newPagesUpdateCmd())

	return cmd
}

func newPagesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List wiki pages for a course",
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

			pages, _, err := canvas.ListPages(cmd.Context(), client, courseID, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "pages.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(pages, "pages.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, p := range pages {
				fmt.Fprintf(w, "%s\t%s\n", p.URL, p.Title)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newPagesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get PAGE_URL",
		Short: "Get a single wiki page with body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			pageURL := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			page, err := canvas.GetPage(cmd.Context(), client, courseID, pageURL)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "pages.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(page, "pages.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "URL:   %s\n", page.URL)
			fmt.Fprintf(w, "Title: %s\n", page.Title)
			if page.Body != "" {
				fmt.Fprintf(w, "Body:\n%s\n", page.Body)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newPagesUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update PAGE_URL",
		Short: "Update a wiki page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			bodyFile, _ := cmd.Flags().GetString("body-file")
			if bodyFile == "" {
				return fmt.Errorf("--body-file is required")
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			pageURL := args[0]

			body, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}

			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/pages/%s", courseID, pageURL)

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:         "PUT",
					Path:           path,
					ResourceIDs:    []string{courseID, pageURL},
					PayloadSummary: fmt.Sprintf("body=%s", truncateString(string(body), 120)),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			updates := map[string]any{
				"body": string(body),
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			_, err = canvas.UpdatePage(cmd.Context(), client, courseID, pageURL, updates)
			if err != nil {
				return err
			}

			writeAudit(cfg, "pages.update", "PUT", path, string(body), false)

			fmt.Fprintf(cmd.OutOrStdout(), "Page %s updated\n", pageURL)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("body-file", "", "path to file with page body (required)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}
