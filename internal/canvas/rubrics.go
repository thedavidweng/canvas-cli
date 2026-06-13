package canvas

import (
	"context"
	"fmt"
)

// ListRubrics returns all rubrics for a course.
// It sends GET /api/v1/courses/{courseID}/rubrics.
func ListRubrics(ctx context.Context, client *Client, courseID string) ([]Rubric, error) {
	rubrics, _, err := List[Rubric](ctx, client, fmt.Sprintf("/api/v1/courses/%s/rubrics", courseID), nil, 100)
	if err != nil {
		return nil, fmt.Errorf("list rubrics for course %s: %w", courseID, err)
	}

	return rubrics, nil
}
