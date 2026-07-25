package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
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

			announcements, _, err := canvas.ListAnnouncements(cmd.Context(), client, courseID, nil)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "announcements.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, announcements, "announcements.list", jsonMode, func(w io.Writer) error {
				for _, a := range announcements {
					fmt.Fprintf(w, "%s\t%s\n", a.ID, a.Title)
				}
				return nil
			})
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			announcementID := args[0]

			topic, err := canvas.GetAnnouncement(cmd.Context(), client, courseID, announcementID)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "announcements.get", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, topic, "announcements.get", jsonMode, func(w io.Writer) error {
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
			})
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

			path := fmt.Sprintf("/api/v1/courses/%s/discussion_topics", courseID)

			spec := MutationSpec{
				Command:        "announcements.create",
				Level:          safety.LowRiskWrite,
				Method:         "POST",
				Path:           path,
				DryRun:         dryRun,
				Confirm:        confirm,
				ResourceIDs:    []string{courseID},
				PayloadSummary: fmt.Sprintf("title=%q is_announcement=true body=%s", title, truncateString(string(body), 120)),
				AuditBody:      string(body),
			}

			return Run(cmd.Context(), cfg, cmd.OutOrStdout(), false, spec,
				func(ctx context.Context, client *canvas.Client) (any, int, error) {
					topic, err := canvas.CreateAnnouncement(ctx, client, courseID, title, string(body))
					if err != nil {
						return nil, 0, err
					}
					return &topic, 200, nil
				},
				func(w io.Writer, data any) error {
					topic, _ := data.(*canvas.DiscussionTopic)
					fmt.Fprintf(w, "Announcement created (ID: %s, title: %s)\n", topic.ID, topic.Title)
					return nil
				},
			)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("title", "", "announcement title (required)")
	cmd.Flags().String("body-file", "", "path to file with announcement body (required)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}
