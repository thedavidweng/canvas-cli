package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// ListEnrollments returns all enrollments for a course.
// It sends GET /api/v1/courses/{courseID}/enrollments with include[]=total_scores.
// The opts parameter controls additional query parameters, pagination limit, and page size.
func ListEnrollments(ctx context.Context, client *Client, courseID string, opts RequestOptions) ([]Enrollment, PaginationMeta, error) {
	query := opts.Query
	if query == nil {
		query = url.Values{}
	}
	query.Set("include[]", "total_scores")

	var enrollments []Enrollment
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/enrollments", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		Limit:      opts.Limit,
		DecodeInto: &enrollments,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list enrollments for course %s: %w", courseID, err)
	}

	return enrollments, meta.Pagination, nil
}
