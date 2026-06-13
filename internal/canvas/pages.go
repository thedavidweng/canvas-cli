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

	var pages []Page
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/pages", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &pages,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list pages for course %s: %w", courseID, err)
	}

	return pages, meta.Pagination, nil
}

// GetPage returns a single page by its URL slug.
// It sends GET /api/v1/courses/{courseID}/pages/{pageURL}.
func GetPage(ctx context.Context, client *Client, courseID, pageURL string) (Page, error) {
	var page Page
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/pages/%s", courseID, pageURL),
		DecodeInto: &page,
	})
	if err != nil {
		return page, fmt.Errorf("get page %s in course %s: %w", pageURL, courseID, err)
	}

	return page, nil
}
