package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// ListSections returns all sections for a course.
// It sends GET /api/v1/courses/{courseID}/sections with include[]=total_students.
func ListSections(ctx context.Context, client *Client, courseID string) ([]Section, error) {
	query := url.Values{
		"include[]": {"total_students"},
	}

	sections, _, err := List[Section](ctx, client, fmt.Sprintf("/api/v1/courses/%s/sections", courseID), query, 100)
	if err != nil {
		return nil, fmt.Errorf("list sections for course %s: %w", courseID, err)
	}

	return sections, nil
}
