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

	var sections []Section
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/sections", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &sections,
	})
	if err != nil {
		return nil, fmt.Errorf("list sections for course %s: %w", courseID, err)
	}

	return sections, nil
}
