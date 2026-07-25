package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewAssignmentsCmd returns the `assignments` parent command with all subcommands.
func NewAssignmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assignments",
		Short: "Manage assignments",
		Long:  `List, get, submit, and manage Canvas assignments.`,
	}

	cmd.AddCommand(newAssignmentsListCmd())
	cmd.AddCommand(newAssignmentsGetCmd())
	cmd.AddCommand(newAssignmentGroupsListCmd())
	cmd.AddCommand(newAssignmentsSubmitCmd())
	cmd.AddCommand(newAssignmentsUpdateCmd())

	return cmd
}

func newAssignmentsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List assignments for a course",
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

			query := url.Values{}
			if bucket, _ := cmd.Flags().GetString("bucket"); bucket != "" {
				query.Set("bucket", bucket)
			}
			if sort, _ := cmd.Flags().GetString("sort"); sort != "" {
				query.Set("sort", sort)
			}
			if order, _ := cmd.Flags().GetString("order"); order != "" {
				query.Set("order", order)
			}
			if search, _ := cmd.Flags().GetString("search"); search != "" {
				query.Set("search", search)
			}
			if dueBefore, _ := cmd.Flags().GetString("due-before"); dueBefore != "" {
				query.Set("due_before", dueBefore)
			}
			if dueAfter, _ := cmd.Flags().GetString("due-after"); dueAfter != "" {
				query.Set("due_after", dueAfter)
			}
			publishedFilter, _ := cmd.Flags().GetString("published")
			if includeSubmission, _ := cmd.Flags().GetBool("include-submission"); includeSubmission {
				query.Set("include[]", "submission")
			}

			client := newClientFromCfg(cfg)
			assignments, _, err := canvas.ListAssignments(cmd.Context(), client, courseID, query)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "assignments.list")
					return writeEnvelope(cmd.OutOrStdout(), cfg, env)
				}
				return err
			}

			if publishedFilter != "" {
				wantPublished := publishedFilter == "true"
				filtered := make([]canvas.Assignment, 0, len(assignments))
				for _, a := range assignments {
					if a.Published == wantPublished {
						filtered = append(filtered, a)
					}
				}
				assignments = filtered
			}

			if jsonMode {
				env := output.NewSuccess(assignments, "assignments.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return writeEnvelope(cmd.OutOrStdout(), cfg, env)
			}

			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Name", "Due At", "Points", "Published"},
			}
			for _, a := range assignments {
				dueAt := "-"
				if a.DueAt != nil {
					dueAt = *a.DueAt
				}
				published := "no"
				if a.Published {
					published = "yes"
				}
				tbl.Rows = append(tbl.Rows, []string{
					a.ID, a.Name, dueAt, formatFloat(a.PointsPossible), published,
				})
			}
			return tbl.Render(w, cfg.OutputNoColor)
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("bucket", "", "filter by bucket (past|overdue|undated|ungraded|unsubmitted|upcoming|future)")
	cmd.Flags().String("sort", "", "sort field (due_at|name|position)")
	cmd.Flags().String("order", "", "sort order (asc|desc)")
	cmd.Flags().String("search", "", "search assignments by name")
	cmd.Flags().String("due-before", "", "filter by due date before")
	cmd.Flags().String("due-after", "", "filter by due date after")
	cmd.Flags().String("published", "", "filter by published state (true|false)")
	cmd.Flags().Bool("include-submission", false, "include submission data")
	return cmd
}

func newAssignmentsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get ASSIGNMENT_ID",
		Short: "Get an assignment by ID",
		Args:  cobra.ExactArgs(1),
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
			assignmentID := args[0]

			client := newClientFromCfg(cfg)
			assignment, err := canvas.GetAssignment(cmd.Context(), client, courseID, assignmentID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "assignments.get")
					return writeEnvelope(cmd.OutOrStdout(), cfg, env)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(assignment, "assignments.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return writeEnvelope(cmd.OutOrStdout(), cfg, env)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:        %s\n", assignment.ID)
			fmt.Fprintf(w, "Name:      %s\n", assignment.Name)
			fmt.Fprintf(w, "Course ID: %s\n", assignment.CourseID)
			fmt.Fprintf(w, "Points:    %s\n", formatFloat(assignment.PointsPossible))
			fmt.Fprintf(w, "Published: %v\n", assignment.Published)
			if assignment.DueAt != nil {
				fmt.Fprintf(w, "Due At:    %s\n", *assignment.DueAt)
			}
			if len(assignment.SubmissionTypes) > 0 {
				fmt.Fprintf(w, "Types:     %s\n", strings.Join(assignment.SubmissionTypes, ", "))
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	return cmd
}

func newAssignmentGroupsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "List assignment groups for a course",
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

			client := newClientFromCfg(cfg)
			groups, err := canvas.ListAssignmentGroups(cmd.Context(), client, courseID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "assignments.groups")
					return writeEnvelope(cmd.OutOrStdout(), cfg, env)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(groups, "assignments.groups", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return writeEnvelope(cmd.OutOrStdout(), cfg, env)
			}

			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Name", "Position", "Weight"},
			}
			for _, g := range groups {
				tbl.Rows = append(tbl.Rows, []string{
					g.ID, g.Name, fmt.Sprintf("%d", g.Position), formatFloat(g.GroupWeight),
				})
			}
			return tbl.Render(w, cfg.OutputNoColor)
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	return cmd
}

func newAssignmentsSubmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit ASSIGNMENT_ID [BODY]",
		Short: "Submit an assignment",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			assignmentID := args[0]
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			textBody, _ := cmd.Flags().GetString("text")
			urlFlag, _ := cmd.Flags().GetString("url")
			filePath, _ := cmd.Flags().GetString("file")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if textBody == "" && urlFlag == "" && filePath == "" && len(args) > 1 {
				textBody = args[1]
			}

			if textBody == "" && urlFlag == "" && filePath == "" {
				return fmt.Errorf("submit requires one of: --text, --url, or --file")
			}

			var submissionType string
			switch {
			case textBody != "":
				submissionType = "online_text_entry"
			case urlFlag != "":
				submissionType = "online_url"
			case filePath != "":
				submissionType = "online_upload"
			}

			client := newClientFromCfg(cfg)

			assignment, err := canvas.GetAssignment(cmd.Context(), client, courseID, assignmentID)
			if err != nil {
				return fmt.Errorf("get assignment: %w", err)
			}

			if len(assignment.SubmissionTypes) > 0 {
				allowed := false
				for _, t := range assignment.SubmissionTypes {
					if t == submissionType {
						allowed = true
						break
					}
				}
				if !allowed {
					return fmt.Errorf("assignment %s does not accept %s submission (allowed: %s)",
						assignmentID, submissionType, strings.Join(assignment.SubmissionTypes, ", "))
				}
			}

			sub := canvas.SubmissionRequest{
				SubmissionType: submissionType,
			}
			switch submissionType {
			case "online_text_entry":
				sub.Body = textBody
			case "online_url":
				sub.URL = urlFlag
			}

			subJSON, _ := json.Marshal(sub)

			auditPath := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions", courseID, assignmentID)

			spec := MutationSpec{
				Command:        "assignments.submit",
				Level:          safety.LowRiskWrite,
				Method:         "POST",
				Path:           auditPath,
				DryRun:         dryRun,
				Confirm:        confirm,
				ResourceIDs:    []string{courseID, assignmentID},
				PayloadSummary: fmt.Sprintf("type=%s body=%s", submissionType, truncateString(string(subJSON), 120)),
				AuditBody:      string(subJSON),
				Resource: map[string]string{
					"course_id":     courseID,
					"assignment_id": assignmentID,
				},
			}

			dryRunShortCircuit, err := CheckAndPreview(cfg, cmd.OutOrStdout(), spec)
			if err != nil {
				return err
			}
			if dryRunShortCircuit {
				return nil
			}

			if submissionType == "online_upload" {
				content, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("read file %s: %w", filePath, err)
				}

				fileID, err := canvas.UploadFile(cmd.Context(), client, courseID, filePath, content)
				if err != nil {
					return fmt.Errorf("upload file: %w", err)
				}

				sub.FileIDs = []string{fileID}
			}

			result, err := canvas.SubmitAssignment(cmd.Context(), client, courseID, assignmentID, sub)
			if err != nil {
				RecordAudit(cfg, spec, 0, false)
				return writeError(cmd.OutOrStdout(), cfg, fmt.Errorf("submit assignment: %w", err), "assignments.submit", jsonMode)
			}

			RecordAudit(cfg, spec, 200, true)

			if jsonMode {
				env := output.NewSuccess(result, "assignments.submit", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return writeEnvelope(cmd.OutOrStdout(), cfg, env)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Submission submitted (ID: %s, state: %s)\n",
				result.ID, result.WorkflowState)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("text", "", "text body for online_text_entry submission")
	cmd.Flags().String("url", "", "URL for online_url submission")
	cmd.Flags().String("file", "", "file path for online_upload submission")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	cmd.Flags().Bool("read-only", false, "block all write operations")
	return cmd
}

func newAssignmentsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update ASSIGNMENT_ID",
		Short: "Update an assignment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			dueAt, _ := cmd.Flags().GetString("due-at")
			if dueAt == "" {
				return fmt.Errorf("--due-at is required")
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			assignmentID := args[0]

			path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s", courseID, assignmentID)
			payload := fmt.Sprintf(`{"assignment":{"due_at":%q}}`, dueAt)

			spec := MutationSpec{
				Command:        "assignments.update",
				Level:          safety.LowRiskWrite,
				Method:         "PUT",
				Path:           path,
				DryRun:         dryRun,
				Confirm:        confirm,
				ResourceIDs:    []string{courseID, assignmentID},
				PayloadSummary: fmt.Sprintf("due_at=%s", dueAt),
				AuditBody:      payload,
			}

			return Run(cmd.Context(), cfg, cmd.OutOrStdout(), false, spec,
				func(ctx context.Context, client *canvas.Client) (any, int, error) {
					updates := map[string]any{"due_at": dueAt}
					_, err := canvas.UpdateAssignment(ctx, client, courseID, assignmentID, updates)
					if err != nil {
						return nil, 0, err
					}
					return nil, 200, nil
				},
				func(w io.Writer, _ any) error {
					fmt.Fprintf(w, "Assignment %s updated (due_at: %s)\n", assignmentID, dueAt)
					return nil
				},
			)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("due-at", "", "new due date (ISO 8601 format, required)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}

// formatFloat formats a float64 for display, removing trailing zeros.
func formatFloat(f float64) string {
	return fmt.Sprintf("%g", f)
}
