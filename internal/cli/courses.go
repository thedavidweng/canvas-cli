package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
func splitCSV(s string) []string {
	var parts []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// NewCoursesCmd returns the `courses` parent command with all subcommands.
func NewCoursesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "courses",
		Short: "Manage courses",
		Long:  `List, get, and manage Canvas courses.`,
	}

	cmd.AddCommand(newCoursesListCmd())
	cmd.AddCommand(newCoursesGetCmd())
	cmd.AddCommand(newCoursesTabsCmd())
	cmd.AddCommand(newCoursesExportContextCmd())
	cmd.AddCommand(newCoursesExportCmd())
	cmd.AddCommand(newCoursesExportsCmd())

	return cmd
}

// newCoursesListCmd returns `courses list`.
func newCoursesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List courses",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")

			query := url.Values{}
			if v, _ := cmd.Flags().GetString("enrollment-state"); v != "" {
				query.Set("enrollment_state", v)
			}
			if v, _ := cmd.Flags().GetString("enrollment-type"); v != "" {
				query.Set("enrollment_type", v)
			}
			if v, _ := cmd.Flags().GetString("state"); v != "" {
				query.Set("state", v)
			}
			if v, _ := cmd.Flags().GetString("include"); v != "" {
				for _, inc := range splitCSV(v) {
					query.Add("include[]", inc)
				}
			}
			if v, _ := cmd.Flags().GetString("search"); v != "" {
				query.Set("search", v)
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			courses, _, err := canvas.ListCourses(cmd.Context(), client, query)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "courses.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(courses, "courses.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode: table output
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Name", "Code", "State"},
			}
			for _, c := range courses {
				tbl.Rows = append(tbl.Rows, []string{c.ID, c.Name, c.CourseCode, c.WorkflowState})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("enrollment-state", "", "filter by enrollment state (active|invited_or_pending|completed)")
	cmd.Flags().String("enrollment-type", "", "filter by enrollment type (teacher|student|ta|observer|designer)")
	cmd.Flags().String("state", "", "filter by state (available|completed|unpublished)")
	cmd.Flags().String("include", "", "include additional fields (comma-separated)")
	cmd.Flags().String("search", "", "search courses by name")
	return cmd
}

// newCoursesGetCmd returns `courses get COURSE_ID`.
func newCoursesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get COURSE_ID",
		Short: "Get a course by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID := args[0]

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			course, err := canvas.GetCourse(cmd.Context(), client, courseID, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "courses.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(course, "courses.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:    %s\n", course.ID)
			fmt.Fprintf(w, "Name:  %s\n", course.Name)
			fmt.Fprintf(w, "Code:  %s\n", course.CourseCode)
			fmt.Fprintf(w, "State: %s\n", course.WorkflowState)
			if course.Term != nil {
				fmt.Fprintf(w, "Term:  %s\n", course.Term.Name)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// newCoursesTabsCmd returns `courses tabs`.
func newCoursesTabsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tabs",
		Short: "List course tabs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			tabs, err := canvas.ListCourseTabs(cmd.Context(), client, courseID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "courses.tabs")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(tabs, "courses.tabs", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Label", "Type", "Position"},
			}
			for _, t := range tabs {
				tbl.Rows = append(tbl.Rows, []string{t.ID, t.Label, t.Type, fmt.Sprintf("%d", t.Position)})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	return cmd
}
