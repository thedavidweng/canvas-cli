package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
)

// ResponseMeta captures metadata from a Canvas API response.
type ResponseMeta struct {
	RateLimit  *RateLimit     `json:"rate_limit,omitempty"`
	Pagination PaginationMeta `json:"pagination"`
	Warnings   []string       `json:"warnings,omitempty"`
}

// Request executes a Canvas API request with optional pagination and decoding.
func Request(ctx context.Context, client *Client, opts *RequestOptions) (*ResponseMeta, error) {
	meta := &ResponseMeta{}

	if opts.Paginate {
		if opts.DecodeInto == nil {
			return meta, fmt.Errorf("decodeInto required when paginate is true")
		}

		pageSize := opts.PageSize
		if pageSize == 0 {
			pageSize = 100
		}

		pagMeta, err := paginateInto(ctx, client, opts.PathOrURL, opts.Query, opts.Limit, pageSize, opts.DecodeInto)
		if err != nil {
			return meta, fmt.Errorf("pagination failed: %w", err)
		}
		meta.Pagination = pagMeta
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

// paginateInto auto-paginates a Canvas list endpoint, decoding each page
// directly into dst (a pointer to a slice) without an intermediate
// marshal/unmarshal round-trip. Each page is decoded into []json.RawMessage
// and individual elements are unmarshaled into the destination slice via
// reflection, preserving the caller's concrete element type.
func paginateInto(ctx context.Context, client *Client, path string, query url.Values, limit, pageSize int, dst any) (PaginationMeta, error) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Pointer || dstVal.Elem().Kind() != reflect.Slice {
		return PaginationMeta{}, fmt.Errorf("paginateInto: dst must be a pointer to slice, got %T", dst)
	}
	sliceVal := dstVal.Elem()
	elemType := sliceVal.Type().Elem()

	meta := PaginationMeta{
		Paginated: true,
		PageSize:  pageSize,
		Limit:     limit,
	}

	if query == nil {
		query = url.Values{}
	}
	if pageSize > 0 {
		query.Set("per_page", fmt.Sprintf("%d", pageSize))
	}

	currentPath := path
	var nextAbsoluteURL string

	for {
		select {
		case <-ctx.Done():
			return meta, ctx.Err()
		default:
		}

		var resp *http.Response
		var err error

		if nextAbsoluteURL != "" {
			resp, err = client.DoURL(ctx, "GET", nextAbsoluteURL, nil)
		} else {
			resp, err = client.Do(ctx, "GET", currentPath, query, nil)
		}
		if err != nil {
			return meta, fmt.Errorf("pagination request failed: %w", err)
		}

		meta.RequestCount++

		if resp.StatusCode >= 400 {
			var baseURL string
			if client.cookie != "" && client.token == "" {
				baseURL = client.baseURL
			}
			env := NormalizeError(resp, "GET", baseURL)
			resp.Body.Close()
			return meta, fmt.Errorf("api error (status %d): %s", env.Error.Status, env.Error.Message)
		}

		var rawItems []json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&rawItems); err != nil {
			resp.Body.Close()
			return meta, fmt.Errorf("failed to decode page: %w", err)
		}
		resp.Body.Close()

		for _, raw := range rawItems {
			if limit > 0 && sliceVal.Len() >= limit {
				break
			}
			elem := reflect.New(elemType)
			if err := json.Unmarshal(raw, elem.Interface()); err != nil {
				return meta, fmt.Errorf("failed to decode item: %w", err)
			}
			sliceVal = reflect.Append(sliceVal, elem.Elem())
		}

		if limit > 0 && sliceVal.Len() >= limit {
			break
		}

		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break
		}

		links := ParseLinkHeader(linkHeader)
		nextURL, ok := links["next"]
		if !ok || nextURL == "" {
			break
		}

		parsed, err := url.Parse(nextURL)
		if err != nil {
			return meta, fmt.Errorf("failed to parse next URL: %w", err)
		}

		if parsed.Scheme != "" && parsed.Host != "" {
			nextAbsoluteURL = nextURL
			currentPath = ""
			query = nil
		} else {
			nextAbsoluteURL = ""
			currentPath = parsed.Path
			if parsed.RawQuery != "" {
				query = parsed.Query()
			} else {
				query = nil
			}
		}
	}

	dstVal.Elem().Set(sliceVal)
	meta.TotalItems = sliceVal.Len()
	return meta, nil
}
