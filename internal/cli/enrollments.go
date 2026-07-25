package cli

import (
	"fmt"
	"io"

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
			ctx := cmd.Context()
			client, err := getClientFromContext(ctx)
			if err != nil {
				return err
			}
			cfg := GetConfig(ctx)

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")

			enrollments, _, err := canvas.ListEnrollments(ctx, client, courseID, canvas.RequestOptions{})
			if err != nil {
				return writeError(cmd.OutOrStdout(), cfg, err, "enrollments.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, enrollments, "enrollments.list", jsonMode, func(w io.Writer) error {
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
				return tbl.Render(w, cfg.OutputNoColor)
			})
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
