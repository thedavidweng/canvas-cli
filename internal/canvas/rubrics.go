package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// ListRubrics returns all rubrics for a course.
// It sends GET /api/v1/courses/{courseID}/rubrics.
func ListRubrics(ctx context.Context, client *Client, courseID string) ([]Rubric, error) {
	query := url.Values{}

	var rubrics []Rubric
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/rubrics", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &rubrics,
	})
	if err != nil {
		return nil, fmt.Errorf("list rubrics for course %s: %w", courseID, err)
	}

	return rubrics, nil
}
