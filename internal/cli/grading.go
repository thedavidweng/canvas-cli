package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewGradingCmd returns the `grade` parent command.
func NewGradingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grade",
		Short: "Manage grading",
		Long:  `Set grades, add comments, and manage rubric assessments.`,
	}

	cmd.AddCommand(newGradeSetCmd())
	cmd.AddCommand(newGradeCommentCmd())
	cmd.AddCommand(newGradeRubricCmd())
	cmd.AddCommand(newGradeImportCmd())

	return cmd
}

func newGradeSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a grade for a submission",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			userID, _ := cmd.Flags().GetString("user")
			score, _ := cmd.Flags().GetString("score")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}
			if userID == "" {
				return fmt.Errorf("--user is required")
			}
			if score == "" {
				return fmt.Errorf("--score is required")
			}

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions/%s", courseID, assignmentID, userID)

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:         "PUT",
					Path:           path,
					ResourceIDs:    []string{courseID, assignmentID, userID},
					PayloadSummary: fmt.Sprintf(`{"submission":{"posted_grade":"%s"}}`, score),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := newClientFromCfg(cfg)
			sub, err := canvas.SetGrade(cmd.Context(), client, courseID, assignmentID, userID, score)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "grade.set")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			writeAudit(cfg, "grade.set", "PUT", path,
				fmt.Sprintf(`{"submission":{"posted_grade":"%s"}}`, score), false)

			if jsonMode {
				env := output.NewSuccess(sub, "grade.set", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Grade set to %s for user %s on assignment %s\n", score, userID, assignmentID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().String("user", "", "user ID (required)")
	cmd.Flags().String("score", "", "grade score (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newGradeCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Add a comment to a submission",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			userID, _ := cmd.Flags().GetString("user")
			comment, _ := cmd.Flags().GetString("comment")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}
			if userID == "" {
				return fmt.Errorf("--user is required")
			}
			if comment == "" {
				return fmt.Errorf("--comment is required")
			}

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions/%s", courseID, assignmentID, userID)

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:         "PUT",
					Path:           path,
					ResourceIDs:    []string{courseID, assignmentID, userID},
					PayloadSummary: fmt.Sprintf(`{"comment":{"text_comment":"%s"}}`, comment),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := newClientFromCfg(cfg)
			sub, err := canvas.AddComment(cmd.Context(), client, courseID, assignmentID, userID, comment)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "grade.comment")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			writeAudit(cfg, "grade.comment", "PUT", path,
				fmt.Sprintf(`{"comment":{"text_comment":"%s"}}`, comment), false)

			if jsonMode {
				env := output.NewSuccess(sub, "grade.comment", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Comment added for user %s on assignment %s\n", userID, assignmentID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().String("user", "", "user ID (required)")
	cmd.Flags().String("comment", "", "comment text (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// GradeImportResult holds the result of a CLI grade import operation.
type GradeImportResult struct {
	Total    int      `json:"total"`
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Warnings []string `json:"warnings,omitempty"`
}

// gradeImportPartialFailureError is returned when some grade imports fail.
type gradeImportPartialFailureError struct {
	msg string
}

func (e *gradeImportPartialFailureError) Error() string { return e.msg }
func (e *gradeImportPartialFailureError) ExitCode() int { return output.CodePartialFailure }

func newGradeImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import grades from a CSV file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			csvPath, _ := cmd.Flags().GetString("csv")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}
			if csvPath == "" {
				return fmt.Errorf("--csv is required")
			}

			// Parse CSV
			gradeData, err := parseGradeCSV(csvPath)
			if err != nil {
				return fmt.Errorf("parse CSV: %w", err)
			}
			if len(gradeData) == 0 {
				return fmt.Errorf("csv file contains no grade data")
			}

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions/update_grades", courseID, assignmentID)

			if dryRun {
				var summary strings.Builder
				for uid, score := range gradeData {
					fmt.Fprintf(&summary, "  user %s -> %s\n", uid, score)
				}
				preview := safety.FormatPreview(safety.Preview{
					Method:         "POST",
					Path:           path,
					ResourceIDs:    []string{courseID, assignmentID},
					PayloadSummary: fmt.Sprintf("%d grades:\n%s", len(gradeData), summary.String()),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			// Warn if --confirm without prior --dry-run (human mode only)
			result := &GradeImportResult{Total: len(gradeData)}
			w := cmd.OutOrStdout()
			if !jsonMode {
				fmt.Fprintf(w, "warning: importing %d grades without prior --dry-run\n", len(gradeData))
			}

			client := newClientFromCfg(cfg)
			subs, err := canvas.ImportGrades(cmd.Context(), client, courseID, assignmentID, gradeData)
			if err != nil {
				result.Failed = len(gradeData)
				writeAudit(cfg, "grade.import", "POST", path, "bulk import", false)

				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "PARTIAL_FAILURE",
						Message:  err.Error(),
						Category: "partial_failure",
					}, "grade.import")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return &gradeImportPartialFailureError{msg: fmt.Sprintf("grade import failed: %v", err)}
			}

			result.Imported = len(subs)
			writeAudit(cfg, "grade.import", "POST", path, "bulk import", false)

			if jsonMode {
				env := output.NewSuccess(result, "grade.import", canvas.Meta{
					Profile:  cfg.Profile,
					BaseURL:  cfg.BaseURL,
					Warnings: result.Warnings,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			fmt.Fprintf(w, "Imported %d grades for assignment %s\n", result.Imported, assignmentID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().String("csv", "", "CSV file path (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	cmd.Flags().Bool("continue-on-error", false, "continue on partial failure")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

func newGradeRubricCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rubric",
		Short: "Submit a rubric assessment for a submission",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			assignmentID, _ := cmd.Flags().GetString("assignment")
			userID, _ := cmd.Flags().GetString("user")
			rubricJSONPath, _ := cmd.Flags().GetString("rubric-json")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")
			jsonMode, _ := cmd.Flags().GetBool("json")

			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if assignmentID == "" {
				return fmt.Errorf("--assignment is required")
			}
			if userID == "" {
				return fmt.Errorf("--user is required")
			}
			if rubricJSONPath == "" {
				return fmt.Errorf("--rubric-json is required")
			}

			// Read and parse rubric assessment JSON
			data, err := os.ReadFile(rubricJSONPath)
			if err != nil {
				return fmt.Errorf("read rubric JSON: %w", err)
			}
			var rubricAssessment map[string]any
			if err := json.Unmarshal(data, &rubricAssessment); err != nil {
				return fmt.Errorf("parse rubric JSON: %w", err)
			}

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions/%s", courseID, assignmentID, userID)

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:         "PUT",
					Path:           path,
					ResourceIDs:    []string{courseID, assignmentID, userID},
					PayloadSummary: fmt.Sprintf("rubric assessment with %d criteria", len(rubricAssessment)),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := newClientFromCfg(cfg)
			sub, err := canvas.GradeRubric(cmd.Context(), client, courseID, assignmentID, userID, rubricAssessment)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "grade.rubric")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			writeAudit(cfg, "grade.rubric", "PUT", path,
				fmt.Sprintf("rubric assessment with %d criteria", len(rubricAssessment)), false)

			if jsonMode {
				env := output.NewSuccess(sub, "grade.rubric", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Rubric assessment submitted for user %s on assignment %s\n", userID, assignmentID)
			return nil
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("assignment", "", "assignment ID (required)")
	cmd.Flags().String("user", "", "user ID (required)")
	cmd.Flags().String("rubric-json", "", "path to rubric assessment JSON file (required)")
	cmd.Flags().Bool("dry-run", false, "preview without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// parseGradeCSV reads a CSV file and returns a map of user_id -> score.
// The CSV must have headers: user_id,score
func parseGradeCSV(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open CSV: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv must have a header row and at least one data row")
	}

	// Find column indices from header
	header := records[0]
	userIDIdx := -1
	scoreIdx := -1
	for i, col := range header {
		switch strings.TrimSpace(strings.ToLower(col)) {
		case "user_id":
			userIDIdx = i
		case "score":
			scoreIdx = i
		}
	}
	if userIDIdx == -1 {
		return nil, fmt.Errorf("csv missing 'user_id' column")
	}
	if scoreIdx == -1 {
		return nil, fmt.Errorf("csv missing 'score' column")
	}

	gradeData := make(map[string]string)
	for _, row := range records[1:] {
		if len(row) <= userIDIdx || len(row) <= scoreIdx {
			continue
		}
		uid := strings.TrimSpace(row[userIDIdx])
		score := strings.TrimSpace(row[scoreIdx])
		if uid != "" && score != "" {
			gradeData[uid] = score
		}
	}

	return gradeData, nil
}
