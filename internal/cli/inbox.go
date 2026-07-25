package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/safety"
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			jsonMode, _ := cmd.Flags().GetBool("json")

			conversations, _, err := canvas.ListConversations(cmd.Context(), client, &canvas.RequestOptions{PageSize: 100})
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "inbox.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, conversations, "inbox.list", jsonMode, func(w io.Writer) error {
				for _, c := range conversations {
					fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, c.WorkflowState, c.Subject)
				}
				return nil
			})
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			conversationID := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")

			conversation, err := canvas.GetConversation(cmd.Context(), client, conversationID)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "inbox.get", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, conversation, "inbox.get", jsonMode, func(w io.Writer) error {
				fmt.Fprintf(w, "ID:       %s\n", conversation.ID)
				fmt.Fprintf(w, "Subject:  %s\n", conversation.Subject)
				fmt.Fprintf(w, "State:    %s\n", conversation.WorkflowState)
				fmt.Fprintf(w, "Messages: %d\n", conversation.MessageCount)
				if conversation.LastMessage != "" {
					fmt.Fprintf(w, "Last:     %s\n", conversation.LastMessage)
				}
				return nil
			})
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

			path := "/api/v1/conversations"

			spec := MutationSpec{
				Command:        "inbox.send",
				Level:          safety.LowRiskWrite,
				Method:         "POST",
				Path:           path,
				DryRun:         dryRun,
				Confirm:        confirm,
				PayloadSummary: fmt.Sprintf("to=%s subject=%q body=%s", to, subject, truncateString(body, 120)),
				AuditBody:      body,
			}

			return Run(cmd.Context(), cfg, cmd.OutOrStdout(), false, &spec,
				func(ctx context.Context, client *canvas.Client) (any, int, error) {
					conversation, err := canvas.SendMessage(ctx, client, []string{to}, subject, body)
					if err != nil {
						return nil, 0, err
					}
					return &conversation, 200, nil
				},
				func(w io.Writer, data any) error {
					conversation, _ := data.(*canvas.Conversation)
					fmt.Fprintf(w, "Message sent (conversation %s)\n", conversation.ID)
					return nil
				},
			)
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

			path := fmt.Sprintf("/api/v1/conversations/%s/add_message", conversationID)

			spec := MutationSpec{
				Command:        "inbox.reply",
				Level:          safety.LowRiskWrite,
				Method:         "POST",
				Path:           path,
				DryRun:         dryRun,
				Confirm:        confirm,
				ResourceIDs:    []string{conversationID},
				PayloadSummary: fmt.Sprintf("body=%s", truncateString(body, 120)),
				AuditBody:      body,
			}

			return Run(cmd.Context(), cfg, cmd.OutOrStdout(), false, &spec,
				func(ctx context.Context, client *canvas.Client) (any, int, error) {
					_, err := canvas.ReplyToConversation(ctx, client, conversationID, body)
					if err != nil {
						return nil, 0, err
					}
					return nil, 200, nil
				},
				func(w io.Writer, _ any) error {
					fmt.Fprintf(w, "Reply sent\n")
					return nil
				},
			)
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

			path := fmt.Sprintf("/api/v1/conversations/%s", conversationID)

			spec := MutationSpec{
				Command:        "inbox.archive",
				Level:          safety.LowRiskWrite,
				Method:         "PUT",
				Path:           path,
				DryRun:         dryRun,
				Confirm:        confirm,
				ResourceIDs:    []string{conversationID},
				PayloadSummary: "workflow_state=archived",
				AuditBody:      "archived",
			}

			return Run(cmd.Context(), cfg, cmd.OutOrStdout(), false, &spec,
				func(ctx context.Context, client *canvas.Client) (any, int, error) {
					if err := canvas.ArchiveConversation(ctx, client, conversationID); err != nil {
						return nil, 0, err
					}
					return nil, 200, nil
				},
				func(w io.Writer, _ any) error {
					fmt.Fprintf(w, "Conversation %s archived\n", conversationID)
					return nil
				},
			)
		},
	}
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}
