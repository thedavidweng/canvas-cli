package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// ListPages returns all pages for a course.
// It sends GET /api/v1/courses/{courseID}/pages.
// The per_page parameter defaults to 100.
func ListPages(ctx context.Context, client *Client, courseID string, query url.Values) ([]Page, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	pages, meta, err := List[Page](ctx, client, fmt.Sprintf("/api/v1/courses/%s/pages", courseID), query, 100)
	if err != nil {
		return nil, meta, fmt.Errorf("list pages for course %s: %w", courseID, err)
	}

	return pages, meta, nil
}

// GetPage returns a single page by its URL slug.
// It sends GET /api/v1/courses/{courseID}/pages/{pageURL}.
func GetPage(ctx context.Context, client *Client, courseID, pageURL string) (Page, error) {
	return Get[Page](ctx, client, fmt.Sprintf("/api/v1/courses/%s/pages/%s", courseID, pageURL))
}
