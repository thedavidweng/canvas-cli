package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// NewUsersCmd returns the `users` parent command.
func NewUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users",
		Long:  `List and manage Canvas users.`,
	}

	cmd.AddCommand(newUsersListCmd())

	return cmd
}

func newUsersListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users in a course",
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
			enrollmentType, _ := cmd.Flags().GetString("enrollment-type")

			query := url.Values{}
			if enrollmentType != "" {
				query.Set("enrollment_type[]", enrollmentType)
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			users, _, err := canvas.ListUsers(cmd.Context(), client, courseID, canvas.RequestOptions{
				Query: query,
			})
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "users.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(users, "users.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode: table output
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Name", "Login ID"},
			}
			for _, u := range users {
				tbl.Rows = append(tbl.Rows, []string{u.ID, u.Name, u.LoginID})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("enrollment-type", "", "filter by enrollment type (student|teacher|ta|observer|designer)")
	return cmd
}
