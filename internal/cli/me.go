package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
)

// NewMeCmd returns the `me` parent command with all subcommands.
func NewMeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Manage your user profile and activity",
		Long:  `View and manage your Canvas user profile, activity, todos, and upcoming events.`,
	}

	cmd.AddCommand(newMeGetCmd())
	cmd.AddCommand(newMeActivityCmd())
	cmd.AddCommand(newMeTodoCmd())
	cmd.AddCommand(newMeUpcomingCmd())

	return cmd
}

// newMeGetCmd returns `me get`.
func newMeGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get current user information",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := getClientFromContext(ctx)
			if err != nil {
				return err
			}
			cfg := GetConfig(ctx)

			jsonMode, _ := cmd.Flags().GetBool("json")

			resp, err := client.Do(ctx, "GET", "/api/v1/users/self", nil, nil)
			if err != nil {
				return writeNetworkError(cmd.OutOrStdout(), cfg, err, "me.get", jsonMode)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				env := canvas.NormalizeError(resp, "me.get", cookieAuthBaseURL(cfg)...)
				if jsonMode {
					return writeEnvelope(cmd.OutOrStdout(), cfg, &env)
				}
				return fmt.Errorf("api error: %s (status %d)", env.Error.Message, resp.StatusCode)
			}

			var user canvas.User
			if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, user, "me.get", jsonMode, func(w io.Writer) error {
				fmt.Fprintf(w, "Name:      %s\n", user.Name)
				fmt.Fprintf(w, "ID:        %s\n", user.ID)
				if user.LoginID != "" {
					fmt.Fprintf(w, "Login ID:  %s\n", user.LoginID)
				}
				if user.Email != nil && *user.Email != "" {
					fmt.Fprintf(w, "Email:     %s\n", *user.Email)
				}
				if user.ShortName != "" {
					fmt.Fprintf(w, "Short Name: %s\n", user.ShortName)
				}
				return nil
			})
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// newMeActivityCmd returns `me activity`.
func newMeActivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity",
		Short: "Show recent activity stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := getClientFromContext(ctx)
			if err != nil {
				return err
			}
			cfg := GetConfig(ctx)

			jsonMode, _ := cmd.Flags().GetBool("json")

			items, err := canvas.GetActivityStream(ctx, client)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "me.activity", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, items, "me.activity", jsonMode, func(w io.Writer) error {
				for _, item := range items {
					fmt.Fprintf(w, "[%s] %s\n", item.Type, item.Title)
					if item.Message != "" {
						fmt.Fprintf(w, "  %s\n", item.Message)
					}
					fmt.Fprintf(w, "  Created: %s\n", item.CreatedAt)
				}
				return nil
			})
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// newMeTodoCmd returns `me todo`.
func newMeTodoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo",
		Short: "Show todo items",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := getClientFromContext(ctx)
			if err != nil {
				return err
			}
			cfg := GetConfig(ctx)

			jsonMode, _ := cmd.Flags().GetBool("json")

			items, err := canvas.GetTodoItems(ctx, client)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "me.todo", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, items, "me.todo", jsonMode, func(w io.Writer) error {
				for _, item := range items {
					dueStr := "no due date"
					if item.DueDate != nil {
						dueStr = *item.DueDate
					}
					fmt.Fprintf(w, "[%s] %s (due: %s)\n", item.Type, item.Title, dueStr)
				}
				return nil
			})
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// newMeUpcomingCmd returns `me upcoming`.
func newMeUpcomingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upcoming",
		Short: "Show upcoming events",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := getClientFromContext(ctx)
			if err != nil {
				return err
			}
			cfg := GetConfig(ctx)

			jsonMode, _ := cmd.Flags().GetBool("json")

			items, err := canvas.GetUpcomingEvents(ctx, client)
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "me.upcoming", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, items, "me.upcoming", jsonMode, func(w io.Writer) error {
				for _, item := range items {
					fmt.Fprintf(w, "[%s] %s\n", item.Type, item.Title)
					fmt.Fprintf(w, "  Start: %s\n", item.StartAt)
					if item.EndAt != "" {
						fmt.Fprintf(w, "  End:   %s\n", item.EndAt)
					}
				}
				return nil
			})
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
