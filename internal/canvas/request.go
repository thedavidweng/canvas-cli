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
		// Use pagination for list endpoints
		if opts.DecodeInto == nil {
			return meta, fmt.Errorf("DecodeInto must be set when Paginate is true")
		}

		limit := opts.Limit
		pageSize := opts.PageSize
		if pageSize == 0 {
			pageSize = 100
		}

		// Paginate returns []T, but DecodeInto is a *[]T
		// We need to handle this carefully
		items, pagMeta, err := Paginate[any](ctx, client, opts.PathOrURL, opts.Query, limit, pageSize)
		if err != nil {
			return meta, fmt.Errorf("pagination failed: %w", err)
		}

		meta.Pagination = pagMeta

		// Decode the collected items into the target
		// Since Paginate already decoded each page, we need to re-encode and decode
		// to get the final result into DecodeInto
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

	// Single request (non-paginated)
	resp, err := client.DoWithHeaders(ctx, opts.Method, opts.PathOrURL, opts.Query, opts.Body, opts.Headers)
	if err != nil {
		return meta, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Capture rate limit meta
	meta.RateLimit = CaptureRateMeta(resp)

	// Check for errors
	if resp.StatusCode >= 400 {
		env := NormalizeError(resp, opts.Method)
		return meta, fmt.Errorf("API error: %s (status %d)", env.Error.Message, env.Error.Status)
	}

	// Decode response if target is provided
	if opts.DecodeInto != nil {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(opts.DecodeInto); err != nil {
			return meta, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return meta, nil
}
