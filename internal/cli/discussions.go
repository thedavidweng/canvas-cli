package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewDiscussionsCmd returns the `discussions` parent command.
func NewDiscussionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discussions",
		Short: "Manage discussions",
		Long:  `List, get, and manage Canvas discussion topics.`,
	}

	cmd.AddCommand(newDiscussionsListCmd())
	cmd.AddCommand(newDiscussionsGetCmd())
	cmd.AddCommand(newDiscussionsEntriesCmd())
	cmd.AddCommand(newDiscussionsReplyCmd())
	cmd.AddCommand(newDiscussionsReplyEntryCmd())
	cmd.AddCommand(newDiscussionsCreateCmd())

	return cmd
}

func newDiscussionsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discussion topics for a course",
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
			client := newClientFromCfg(cfg)

			discussions, _, err := canvas.ListDiscussions(cmd.Context(), client, courseID, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "discussions.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(discussions, "discussions.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, d := range discussions {
				fmt.Fprintf(w, "%s\t%s\n", d.ID, d.Title)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newDiscussionsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get DISCUSSION_ID",
		Short: "Get a single discussion topic",
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

			discussionID := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")
			client := newClientFromCfg(cfg)

			topic, err := canvas.GetDiscussion(cmd.Context(), client, courseID, discussionID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "discussions.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(topic, "discussions.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:    %s\n", topic.ID)
			fmt.Fprintf(w, "Title: %s\n", topic.Title)
			if topic.Message != "" {
				fmt.Fprintf(w, "Message: %s\n", topic.Message)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newDiscussionsEntriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entries DISCUSSION_ID",
		Short: "List entries (replies) for a discussion topic",
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

			discussionID := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")
			client := newClientFromCfg(cfg)

			entries, _, err := canvas.ListDiscussionEntries(cmd.Context(), client, courseID, discussionID, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "discussions.entries")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(entries, "discussions.entries", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, e := range entries {
				name := e.UserName
				if name == "" {
					name = e.UserID
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", e.ID, name, e.Message)
			}
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newDiscussionsReplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reply",
		Short: "Reply to a discussion topic",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			did, _ := cmd.Flags().GetString("did")
			message, _ := cmd.Flags().GetString("message")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if did == "" {
				return fmt.Errorf("--did is required")
			}
			if message == "" {
				return fmt.Errorf("--message is required")
			}

			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/discussion_topics/%s/entries", courseID, did)

			if dryRun {
				w := cmd.OutOrStdout()
				fmt.Fprintf(w, "DRY RUN: would send POST %s\n", path)
				fmt.Fprintf(w, "Body: {\"message\": %q}\n", message)
				return nil
			}

			client := newClientFromCfg(cfg)
			entry, err := canvas.ReplyToDiscussion(cmd.Context(), client, courseID, did, message)
			if err != nil {
				return err
			}

			writeAudit(cfg, "discussions.reply", "POST", path, message, false)

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Reply posted (entry %s)\n", entry.ID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("did", "", "discussion topic ID (required)")
	cmd.Flags().String("message", "", "reply message body (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}

func newDiscussionsReplyEntryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reply-entry",
		Short: "Reply to a discussion entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			did, _ := cmd.Flags().GetString("did")
			entryID, _ := cmd.Flags().GetString("entry")
			message, _ := cmd.Flags().GetString("message")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if did == "" {
				return fmt.Errorf("--did is required")
			}
			if entryID == "" {
				return fmt.Errorf("--entry is required")
			}
			if message == "" {
				return fmt.Errorf("--message is required")
			}

			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/discussion_topics/%s/entries/%s/replies", courseID, did, entryID)

			if dryRun {
				w := cmd.OutOrStdout()
				fmt.Fprintf(w, "DRY RUN: would send POST %s\n", path)
				fmt.Fprintf(w, "Body: {\"message\": %q}\n", message)
				return nil
			}

			client := newClientFromCfg(cfg)
			entry, err := canvas.ReplyToEntry(cmd.Context(), client, courseID, did, entryID, message)
			if err != nil {
				return err
			}

			writeAudit(cfg, "discussions.reply-entry", "POST", path, message, false)

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Reply posted (entry %s)\n", entry.ID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("did", "", "discussion topic ID (required)")
	cmd.Flags().String("entry", "", "entry ID to reply to (required)")
	cmd.Flags().String("message", "", "reply message body (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}

func newDiscussionsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a discussion topic for a course",
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
					PayloadSummary: fmt.Sprintf("title=%q body=%s",
						title, truncateString(string(body), 120)),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := newClientFromCfg(cfg)
			topic, err := canvas.CreateDiscussion(cmd.Context(), client, courseID, title, string(body))
			if err != nil {
				return err
			}

			writeAudit(cfg, "discussions.create", "POST", path, string(body), false)

			fmt.Fprintf(cmd.OutOrStdout(), "Discussion created (ID: %s, title: %s)\n", topic.ID, topic.Title)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("title", "", "discussion title (required)")
	cmd.Flags().String("body-file", "", "path to file with discussion body (required)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}
