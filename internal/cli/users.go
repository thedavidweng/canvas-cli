package cli

import (
	"fmt"
	"io"
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
			enrollmentType, _ := cmd.Flags().GetString("enrollment-type")

			query := url.Values{}
			if enrollmentType != "" {
				query.Set("enrollment_type[]", enrollmentType)
			}

			users, _, err := canvas.ListUsers(cmd.Context(), client, courseID, &canvas.RequestOptions{
				Query: query,
			})
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "users.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, users, "users.list", jsonMode, func(w io.Writer) error {
				tbl := output.Table{
					Headers: []string{"ID", "Name", "Login ID"},
				}
				for _, u := range users {
					tbl.Rows = append(tbl.Rows, []string{u.ID, u.Name, u.LoginID})
				}
				return tbl.Render(w, cfg.OutputNoColor)
			})
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("enrollment-type", "", "filter by enrollment type (student|teacher|ta|observer|designer)")
	return cmd
}
