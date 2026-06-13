package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// SetGrade sets a posted grade for a specific user's submission.
// It sends PUT /api/v1/courses/{courseID}/assignments/{assignmentID}/submissions/{userID}
// with submission.posted_grade set to the given score.
func SetGrade(ctx context.Context, client *Client, courseID, assignmentID, userID, score string) (Submission, error) {
	var sub Submission

	payload := map[string]any{
		"submission": map[string]any{
			"posted_grade": score,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return sub, fmt.Errorf("marshal grade request: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method: "PUT",
		PathOrURL: fmt.Sprintf(
			"/api/v1/courses/%s/assignments/%s/submissions/%s",
			courseID, assignmentID, userID,
		),
		Body:       bytes.NewReader(body),
		DecodeInto: &sub,
	})
	if err != nil {
		return sub, fmt.Errorf("set grade for user %s assignment %s in course %s: %w", userID, assignmentID, courseID, err)
	}

	return sub, nil
}

// AddComment adds a text comment to a specific user's submission.
// It sends PUT /api/v1/courses/{courseID}/assignments/{assignmentID}/submissions/{userID}
// with comment.text_comment set to the given comment.
func AddComment(ctx context.Context, client *Client, courseID, assignmentID, userID, comment string) (Submission, error) {
	var sub Submission

	payload := map[string]any{
		"comment": map[string]any{
			"text_comment": comment,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return sub, fmt.Errorf("marshal comment request: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method: "PUT",
		PathOrURL: fmt.Sprintf(
			"/api/v1/courses/%s/assignments/%s/submissions/%s",
			courseID, assignmentID, userID,
		),
		Body:       bytes.NewReader(body),
		DecodeInto: &sub,
	})
	if err != nil {
		return sub, fmt.Errorf("add comment for user %s assignment %s in course %s: %w", userID, assignmentID, courseID, err)
	}

	return sub, nil
}

// GradeRubric submits a rubric assessment for a specific user's submission.
// It sends PUT /api/v1/courses/{courseID}/assignments/{assignmentID}/submissions/{userID}
// with rubric_assessment data.
func GradeRubric(ctx context.Context, client *Client, courseID, assignmentID, userID string, rubricAssessment map[string]any) (Submission, error) {
	var sub Submission
	payload := map[string]any{
		"rubric_assessment": rubricAssessment,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return sub, fmt.Errorf("marshal rubric assessment: %w", err)
	}
	_, err = Request(ctx, client, RequestOptions{
		Method: "PUT",
		PathOrURL: fmt.Sprintf(
			"/api/v1/courses/%s/assignments/%s/submissions/%s",
			courseID, assignmentID, userID,
		),
		Body:       bytes.NewReader(body),
		DecodeInto: &sub,
	})
	if err != nil {
		return sub, fmt.Errorf("grade rubric for user %s assignment %s in course %s: %w", userID, assignmentID, courseID, err)
	}
	return sub, nil
}

// GradeImportResult holds the result of a bulk grade import.
type GradeImportResult struct {
	Submissions []Submission `json:"submissions"`
}

// ImportGrades posts a batch of grades for an assignment.
// It sends POST /api/v1/courses/{courseID}/assignments/{assignmentID}/submissions/update_grades
// with grade_data mapping user IDs to posted_grade values.
func ImportGrades(ctx context.Context, client *Client, courseID, assignmentID string, gradeData map[string]string) ([]Submission, error) {
	payload := map[string]any{
		"grade_data": gradeData,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal grade import request: %w", err)
	}

	var submissions []Submission
	_, err = Request(ctx, client, RequestOptions{
		Method: "POST",
		PathOrURL: fmt.Sprintf(
			"/api/v1/courses/%s/assignments/%s/submissions/update_grades",
			courseID, assignmentID,
		),
		Body:       bytes.NewReader(body),
		DecodeInto: &submissions,
	})
	if err != nil {
		return nil, fmt.Errorf("import grades for assignment %s in course %s: %w", assignmentID, courseID, err)
	}

	return submissions, nil
}
