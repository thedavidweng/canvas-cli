package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// List fetches a paginated list of items from the Canvas API.
// It wraps Request with Paginate=true and decodes the response into []T.
func List[T any](ctx context.Context, client *Client, path string, query url.Values, pageSize int) ([]T, PaginationMeta, error) {
	var items []T
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  path,
		Query:      query,
		Paginate:   true,
		PageSize:   pageSize,
		DecodeInto: &items,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list %s: %w", path, err)
	}

	return items, meta.Pagination, nil
}

// Get fetches a single item from the Canvas API.
// It wraps Request without pagination and decodes the response into T.
func Get[T any](ctx context.Context, client *Client, path string) (T, error) {
	var item T
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  path,
		DecodeInto: &item,
	})
	if err != nil {
		return item, fmt.Errorf("get %s: %w", path, err)
	}

	return item, nil
}
