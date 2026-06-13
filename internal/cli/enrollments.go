package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// NewEnrollmentsCmd returns the `enrollments` parent command.
func NewEnrollmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enrollments",
		Short: "Manage enrollments",
		Long:  `List and manage Canvas course enrollments.`,
	}

	cmd.AddCommand(newEnrollmentsListCmd())

	return cmd
}

func newEnrollmentsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List enrollments for a course",
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

			enrollments, _, err := canvas.ListEnrollments(cmd.Context(), client, courseID, canvas.RequestOptions{})
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "enrollments.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(enrollments, "enrollments.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode: table with name, role, grades
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"Name", "Role", "Current Score", "Current Grade"},
			}
			for _, e := range enrollments {
				name := ""
				if e.User != nil {
					name = e.User.Name
				}
				score := ""
				grade := ""
				if e.Grades != nil {
					if e.Grades.CurrentScore != nil {
						score = fmt.Sprintf("%.1f", *e.Grades.CurrentScore)
					}
					if e.Grades.CurrentGrade != nil {
						grade = *e.Grades.CurrentGrade
					}
				}
				tbl.Rows = append(tbl.Rows, []string{name, e.Role, score, grade})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
