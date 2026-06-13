package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewAnnouncementsCmd returns the `announcements` parent command.
func NewAnnouncementsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "announcements",
		Short: "Manage announcements",
		Long:  `List, get, and create Canvas announcements.`,
	}

	cmd.AddCommand(newAnnouncementsListCmd())
	cmd.AddCommand(newAnnouncementsGetCmd())
	cmd.AddCommand(newAnnouncementsCreateCmd())

	return cmd
}

func newAnnouncementsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List announcements for a course",
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

			announcements, _, err := canvas.ListAnnouncements(cmd.Context(), client, courseID, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "announcements.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(announcements, "announcements.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, a := range announcements {
				fmt.Fprintf(w, "%s\t%s\n", a.ID, a.Title)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newAnnouncementsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get ANNOUNCEMENT_ID",
		Short: "Get an announcement by ID",
		Args:  cobra.ExactArgs(1),
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
			announcementID := args[0]

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			topic, err := canvas.GetAnnouncement(cmd.Context(), client, courseID, announcementID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "announcements.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(topic, "announcements.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:       %s\n", topic.ID)
			fmt.Fprintf(w, "Title:    %s\n", topic.Title)
			fmt.Fprintf(w, "Message:  %s\n", topic.Message)
			postedAt := "n/a"
			if topic.PostedAt != nil {
				postedAt = *topic.PostedAt
			}
			fmt.Fprintf(w, "PostedAt: %s\n", postedAt)
			fmt.Fprintf(w, "UserName: %s\n", topic.UserName)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newAnnouncementsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an announcement for a course",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			title, _ := cmd.Flags().GetString("title")
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			bodyFile, _ := cmd.Flags().GetString("body-file")
			if bodyFile == "" {
				return fmt.Errorf("--body-file is required")
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			body, err := os.ReadFile(bodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}

			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/discussion_topics", courseID)

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:      "POST",
					Path:        path,
					ResourceIDs: []string{courseID},
					PayloadSummary: fmt.Sprintf("title=%q is_announcement=true body=%s",
						title, truncateString(string(body), 120)),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			topic, err := canvas.CreateAnnouncement(cmd.Context(), client, courseID, title, string(body))
			if err != nil {
				return err
			}

			writeAudit(cfg, "announcements.create", "POST", path, string(body), false)

			fmt.Fprintf(cmd.OutOrStdout(), "Announcement created (ID: %s, title: %s)\n", topic.ID, topic.Title)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("title", "", "announcement title (required)")
	cmd.Flags().String("body-file", "", "path to file with announcement body (required)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}
