package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
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
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			resp, err := client.Do(cmd.Context(), "GET", "/api/v1/users/self", nil, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_NETWORK_ERROR",
						Message:  err.Error(),
						Category: "network",
					}, "me.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("failed to reach API: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				env := canvas.NormalizeError(resp, "me.get")
				if jsonMode {
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("API error: %s (status %d)", env.Error.Message, resp.StatusCode)
			}

			var user canvas.User
			if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			if jsonMode {
				env := output.NewSuccess(user, "me.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
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
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			items, err := canvas.GetActivityStream(cmd.Context(), client)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "me.activity")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(items, "me.activity", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, item := range items {
				fmt.Fprintf(w, "[%s] %s\n", item.Type, item.Title)
				if item.Message != "" {
					fmt.Fprintf(w, "  %s\n", item.Message)
				}
				fmt.Fprintf(w, "  Created: %s\n", item.CreatedAt)
			}
			return nil
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
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			items, err := canvas.GetTodoItems(cmd.Context(), client)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "me.todo")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(items, "me.todo", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, item := range items {
				dueStr := "no due date"
				if item.DueDate != nil {
					dueStr = *item.DueDate
				}
				fmt.Fprintf(w, "[%s] %s (due: %s)\n", item.Type, item.Title, dueStr)
			}
			return nil
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
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			items, err := canvas.GetUpcomingEvents(cmd.Context(), client)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "me.upcoming")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(items, "me.upcoming", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, item := range items {
				fmt.Fprintf(w, "[%s] %s\n", item.Type, item.Title)
				fmt.Fprintf(w, "  Start: %s\n", item.StartAt)
				if item.EndAt != "" {
					fmt.Fprintf(w, "  End:   %s\n", item.EndAt)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
