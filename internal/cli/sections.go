package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// NewSectionsCmd returns the `sections` parent command.
func NewSectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sections",
		Short: "Manage sections",
		Long:  `List and manage Canvas course sections.`,
	}

	cmd.AddCommand(newSectionsListCmd())

	return cmd
}

func newSectionsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sections for a course",
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

			sections, err := canvas.ListSections(cmd.Context(), client, courseID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "sections.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(sections, "sections.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode: table output
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Name", "Course ID", "Total Students"},
			}
			for _, s := range sections {
				total := ""
				if s.TotalStudents != nil {
					total = fmt.Sprintf("%d", *s.TotalStudents)
				}
				tbl.Rows = append(tbl.Rows, []string{s.ID, s.Name, s.CourseID, total})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
