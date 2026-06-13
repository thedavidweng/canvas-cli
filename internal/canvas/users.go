package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// ListUsers returns all users in a course filtered by enrollment type.
// It sends GET /api/v1/courses/{courseID}/users with enrollment_type[]=student.
// The opts parameter controls additional query parameters, pagination limit, and page size.
func ListUsers(ctx context.Context, client *Client, courseID string, opts RequestOptions) ([]User, PaginationMeta, error) {
	query := opts.Query
	if query == nil {
		query = url.Values{}
	}
	if query.Get("enrollment_type[]") == "" {
		query.Set("enrollment_type[]", "student")
	}

	var users []User
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/users", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		Limit:      opts.Limit,
		DecodeInto: &users,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list users for course %s: %w", courseID, err)
	}

	return users, meta.Pagination, nil
}
