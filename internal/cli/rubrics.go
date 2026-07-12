package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// NewRubricsCmd returns the `rubrics` parent command.
func NewRubricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rubrics",
		Short: "Manage rubrics",
		Long:  `List and manage Canvas rubrics.`,
	}

	cmd.AddCommand(newRubricsListCmd())

	return cmd
}

func newRubricsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List rubrics for a course",
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

			rubrics, err := canvas.ListRubrics(ctx, client, courseID)
			if err != nil {
				return writeError(cmd.OutOrStdout(), err, "rubrics.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, rubrics, "rubrics.list", jsonMode, func(w io.Writer) error {
				// Human mode: table output
				tbl := output.Table{
					Headers: []string{"ID", "Title", "Points Possible"},
				}
				for _, r := range rubrics {
					tbl.Rows = append(tbl.Rows, []string{r.ID, r.Title, fmt.Sprintf("%.0f", r.PointsPossible)})
				}
				return tbl.Render(w, false)
			})
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
