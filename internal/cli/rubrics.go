package cli

import (
	"fmt"

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
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			client := newClientFromCfg(cfg)

			rubrics, err := canvas.ListRubrics(cmd.Context(), client, courseID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "rubrics.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(rubrics, "rubrics.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode: table output
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Title", "Points Possible"},
			}
			for _, r := range rubrics {
				tbl.Rows = append(tbl.Rows, []string{r.ID, r.Title, fmt.Sprintf("%.0f", r.PointsPossible)})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}
