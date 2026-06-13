package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// SubmissionRequest holds the parameters for submitting an assignment.
type SubmissionRequest struct {
	SubmissionType string   `json:"submission_type"`
	Body           string   `json:"body,omitempty"`
	URL            string   `json:"url,omitempty"`
	FileIDs        []string `json:"file_ids,omitempty"`
}

// validSubmissionTypes is the set of submission types accepted by this client.
var validSubmissionTypes = map[string]bool{
	"online_text_entry": true,
	"online_url":        true,
	"online_upload":     true,
}

// GetSubmission returns a single submission for a specific user and assignment.
// It sends GET /api/v1/courses/{courseID}/assignments/{assignmentID}/submissions/{userID}.
func GetSubmission(ctx context.Context, client *Client, courseID, assignmentID, userID string) (Submission, error) {
	var sub Submission
	_, err := Request(ctx, client, RequestOptions{
		Method: "GET",
		PathOrURL: fmt.Sprintf(
			"/api/v1/courses/%s/assignments/%s/submissions/%s",
			courseID, assignmentID, userID,
		),
		DecodeInto: &sub,
	})
	if err != nil {
		return sub, fmt.Errorf("get submission for user %s assignment %s in course %s: %w", userID, assignmentID, courseID, err)
	}

	return sub, nil
}

// ListSubmissions returns all submissions for an assignment in a course.
// It sends GET /api/v1/courses/{courseID}/assignments/{assignmentID}/submissions with include[]=user.
// The opts parameter controls additional query parameters, pagination limit, and page size.
func ListSubmissions(ctx context.Context, client *Client, courseID, assignmentID string, opts RequestOptions) ([]Submission, PaginationMeta, error) {
	query := opts.Query
	if query == nil {
		query = url.Values{}
	}
	query.Set("include[]", "user")

	var submissions []Submission
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions", courseID, assignmentID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		Limit:      opts.Limit,
		DecodeInto: &submissions,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list submissions for assignment %s in course %s: %w", assignmentID, courseID, err)
	}

	return submissions, meta.Pagination, nil
}

// SubmitAssignment posts a submission for an assignment.
// It sends POST /api/v1/courses/{courseID}/assignments/{assignmentID}/submissions.
// The submission_type must be one of: online_text_entry, online_url, online_upload.
func SubmitAssignment(ctx context.Context, client *Client, courseID, assignmentID string, sub SubmissionRequest) (Submission, error) {
	var result Submission

	if !validSubmissionTypes[sub.SubmissionType] {
		return result, fmt.Errorf("invalid submission type %q: must be one of online_text_entry, online_url, online_upload", sub.SubmissionType)
	}

	payload := map[string]any{
		"submission": sub,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return result, fmt.Errorf("marshal submission request: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:    "POST",
		PathOrURL: fmt.Sprintf("/api/v1/courses/%s/assignments/%s/submissions", courseID, assignmentID),
		Body:      bytes.NewReader(body),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		DecodeInto: &result,
	})
	if err != nil {
		return result, fmt.Errorf("submit assignment %s in course %s: %w", assignmentID, courseID, err)
	}

	return result, nil
}
