package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// NewInboxCmd returns the `inbox` parent command.
func NewInboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Manage inbox conversations",
		Long:  `List, get, send, and manage Canvas inbox conversations.`,
	}

	cmd.AddCommand(newInboxListCmd())
	cmd.AddCommand(newInboxGetCmd())
	cmd.AddCommand(newInboxSendCmd())
	cmd.AddCommand(newInboxReplyCmd())
	cmd.AddCommand(newInboxArchiveCmd())

	return cmd
}

func newInboxListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List inbox conversations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			conversations, _, err := canvas.ListConversations(cmd.Context(), client, canvas.RequestOptions{PageSize: 100})
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "inbox.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(conversations, "inbox.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, c := range conversations {
				fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, c.WorkflowState, c.Subject)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newInboxGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get CONVERSATION_ID",
		Short: "Get a single conversation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			conversationID := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			conversation, err := canvas.GetConversation(cmd.Context(), client, conversationID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "inbox.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(conversation, "inbox.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:       %s\n", conversation.ID)
			fmt.Fprintf(w, "Subject:  %s\n", conversation.Subject)
			fmt.Fprintf(w, "State:    %s\n", conversation.WorkflowState)
			fmt.Fprintf(w, "Messages: %d\n", conversation.MessageCount)
			if conversation.LastMessage != "" {
				fmt.Fprintf(w, "Last:     %s\n", conversation.LastMessage)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newInboxSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a new inbox message",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			to, _ := cmd.Flags().GetString("to")
			subject, _ := cmd.Flags().GetString("subject")
			body, _ := cmd.Flags().GetString("body")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if to == "" {
				return fmt.Errorf("--to is required")
			}
			if subject == "" {
				return fmt.Errorf("--subject is required")
			}
			if body == "" {
				return fmt.Errorf("--body is required")
			}

			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := "/api/v1/conversations"

			if dryRun {
				w := cmd.OutOrStdout()
				fmt.Fprintf(w, "DRY RUN: would send POST %s\n", path)
				fmt.Fprintf(w, "To: %s\n", to)
				fmt.Fprintf(w, "Subject: %s\n", subject)
				fmt.Fprintf(w, "Body: %s\n", body)
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			conversation, err := canvas.SendMessage(cmd.Context(), client, []string{to}, subject, body)
			if err != nil {
				return err
			}

			writeAudit(cfg, "inbox.send", "POST", path, body, false)

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Message sent (conversation %s)\n", conversation.ID)
			return nil
		},
	}
	cmd.Flags().String("to", "", "recipient user ID (required)")
	cmd.Flags().String("subject", "", "message subject (required)")
	cmd.Flags().String("body", "", "message body (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}

func newInboxReplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reply CONVERSATION_ID",
		Short: "Reply to an inbox conversation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			conversationID := args[0]
			body, _ := cmd.Flags().GetString("body")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if body == "" {
				return fmt.Errorf("--body is required")
			}

			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/conversations/%s/add_message", conversationID)

			if dryRun {
				w := cmd.OutOrStdout()
				fmt.Fprintf(w, "DRY RUN: would send POST %s\n", path)
				fmt.Fprintf(w, "Body: %s\n", body)
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			_, err := canvas.ReplyToConversation(cmd.Context(), client, conversationID, body)
			if err != nil {
				return err
			}

			writeAudit(cfg, "inbox.reply", "POST", path, body, false)

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Reply sent\n")
			return nil
		},
	}
	cmd.Flags().String("body", "", "reply message body (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}

func newInboxArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive CONVERSATION_ID",
		Short: "Archive an inbox conversation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			conversationID := args[0]
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if err := checkSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/conversations/%s", conversationID)

			if dryRun {
				w := cmd.OutOrStdout()
				fmt.Fprintf(w, "DRY RUN: would send PUT %s\n", path)
				fmt.Fprintf(w, "workflow_state: archived\n")
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			if err := canvas.ArchiveConversation(cmd.Context(), client, conversationID); err != nil {
				return err
			}

			writeAudit(cfg, "inbox.archive", "PUT", path, "archived", false)

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Conversation %s archived\n", conversationID)
			return nil
		},
	}
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}
