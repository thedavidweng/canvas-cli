package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// ListAssignments returns all assignments for a course.
// It sends GET /api/v1/courses/{courseID}/assignments.
func ListAssignments(ctx context.Context, client *Client, courseID string, query url.Values) ([]Assignment, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	var assignments []Assignment
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/assignments", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &assignments,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list assignments for course %s: %w", courseID, err)
	}

	return assignments, meta.Pagination, nil
}

// GetAssignment returns a single assignment by ID.
// It sends GET /api/v1/courses/{courseID}/assignments/{assignmentID}.
func GetAssignment(ctx context.Context, client *Client, courseID, assignmentID string) (Assignment, error) {
	var assignment Assignment
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/assignments/%s", courseID, assignmentID),
		DecodeInto: &assignment,
	})
	if err != nil {
		return assignment, fmt.Errorf("get assignment %s in course %s: %w", assignmentID, courseID, err)
	}

	return assignment, nil
}

// ListAssignmentGroups returns all assignment groups for a course.
// It sends GET /api/v1/courses/{courseID}/assignment_groups with include[]=assignments.
func ListAssignmentGroups(ctx context.Context, client *Client, courseID string) ([]AssignmentGroup, error) {
	query := url.Values{
		"include[]": {"assignments"},
	}

	var groups []AssignmentGroup
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/assignment_groups", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &groups,
	})
	if err != nil {
		return nil, fmt.Errorf("list assignment groups for course %s: %w", courseID, err)
	}

	return groups, nil
}
