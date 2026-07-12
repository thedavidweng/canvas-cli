package cli

import (
	"fmt"
	"io"

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

			sections, err := canvas.ListSections(ctx, client, courseID)
			if err != nil {
				return writeError(cmd.OutOrStdout(), err, "sections.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, sections, "sections.list", jsonMode, func(w io.Writer) error {
				// Human mode: table output
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
			})
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
