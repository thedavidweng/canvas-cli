package canvas

import (
	"context"
	"encoding/json"
	"fmt"
)

// ResponseMeta captures metadata from a Canvas API response.
type ResponseMeta struct {
	RateLimit  *RateLimit     `json:"rate_limit,omitempty"`
	Pagination PaginationMeta `json:"pagination"`
	Warnings   []string       `json:"warnings,omitempty"`
}

// Request executes a Canvas API request with optional pagination and decoding.
func Request(ctx context.Context, client *Client, opts RequestOptions) (*ResponseMeta, error) {
	meta := &ResponseMeta{}

	if opts.Paginate {
		if opts.DecodeInto == nil {
			return meta, fmt.Errorf("decodeInto required when paginate is true")
		}

		limit := opts.Limit
		pageSize := opts.PageSize
		if pageSize == 0 {
			pageSize = 100
		}

		items, pagMeta, err := Paginate[any](ctx, client, opts.PathOrURL, opts.Query, limit, pageSize)
		if err != nil {
			return meta, fmt.Errorf("pagination failed: %w", err)
		}

		meta.Pagination = pagMeta

		// Paginate decodes each page individually; re-marshal to fit DecodeInto.
		if len(items) > 0 {
			data, err := json.Marshal(items)
			if err != nil {
				return meta, fmt.Errorf("failed to marshal paginated items: %w", err)
			}
			if err := json.Unmarshal(data, opts.DecodeInto); err != nil {
				return meta, fmt.Errorf("failed to decode paginated items: %w", err)
			}
		}

		return meta, nil
	}

	resp, err := client.DoWithHeaders(ctx, opts.Method, opts.PathOrURL, opts.Query, opts.Body, opts.Headers)
	if err != nil {
		return meta, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	meta.RateLimit = CaptureRateMeta(resp)

	if resp.StatusCode >= 400 {
		env := NormalizeError(resp, opts.Method)
		return meta, fmt.Errorf("api error: %s (status %d)", env.Error.Message, env.Error.Status)
	}

	if opts.DecodeInto != nil {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(opts.DecodeInto); err != nil {
			return meta, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return meta, nil
}
