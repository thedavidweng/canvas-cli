package canvas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PaginationMeta captures pagination statistics for a request.
type PaginationMeta struct {
	Paginated    bool `json:"paginated"`
	PageSize     int  `json:"page_size"`
	Limit        int  `json:"limit"`
	RequestCount int  `json:"request_count"`
	TotalItems   int  `json:"total_items"`
}

// ParseLinkHeader extracts rel=URL pairs from a Canvas Link header.
// It parses the standard RFC 5988 format: <URL>; rel="next", <URL>; rel="prev"
// Keys are normalized to lowercase.
func ParseLinkHeader(header string) map[string]string {
	links := make(map[string]string)
	if header == "" {
		return links
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Split into URL and rel
		sections := strings.Split(part, ";")
		if len(sections) < 2 {
			continue
		}

		// Extract URL from <...>
		urlPart := strings.TrimSpace(sections[0])
		if !strings.HasPrefix(urlPart, "<") || !strings.HasSuffix(urlPart, ">") {
			continue
		}
		linkURL := urlPart[1 : len(urlPart)-1]

		// Extract rel value
		for _, s := range sections[1:] {
			s = strings.TrimSpace(s)
			if strings.HasPrefix(s, "rel=") {
				rel := strings.TrimPrefix(s, "rel=")
				rel = strings.Trim(rel, `"`)
				rel = strings.ToLower(rel)
				links[rel] = linkURL
			}
		}
	}

	return links
}

// Paginate auto-paginates a Canvas list endpoint and decodes all items.
// If limit > 0, it stops after collecting that many items.
// pageSize controls the per_page query parameter.
func Paginate[T any](ctx context.Context, client *Client, path string, query url.Values, limit, pageSize int) ([]T, PaginationMeta, error) {
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

	var allItems []T
	currentPath := path
	var nextAbsoluteURL string // non-empty when following a Canvas Link header

	for {
		select {
		case <-ctx.Done():
			return allItems, meta, ctx.Err()
		default:
		}

		var resp *http.Response
		var err error

		// If we have an absolute next URL from a Link header, use DoURL
		// to treat it as opaque (no base URL prepending).
		if nextAbsoluteURL != "" {
			resp, err = client.DoURL(ctx, "GET", nextAbsoluteURL, nil)
		} else {
			resp, err = client.Do(ctx, "GET", currentPath, query, nil)
		}
		if err != nil {
			return allItems, meta, fmt.Errorf("pagination request failed: %w", err)
		}

		meta.RequestCount++

		// Check for error status before decoding body. Canvas may return
		// a JSON error object that would fail to decode as an array.
		if resp.StatusCode >= 400 {
			env := NormalizeError(resp, "GET")
			resp.Body.Close()
			return allItems, meta, fmt.Errorf("API error (status %d): %s", env.Error.Status, env.Error.Message)
		}

		var pageItems []T
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&pageItems); err != nil {
			resp.Body.Close()
			return allItems, meta, fmt.Errorf("failed to decode page: %w", err)
		}
		resp.Body.Close()

		if limit > 0 {
			remaining := limit - len(allItems)
			if remaining <= 0 {
				break
			}
			if len(pageItems) > remaining {
				pageItems = pageItems[:remaining]
			}
		}

		allItems = append(allItems, pageItems...)

		if limit > 0 && len(allItems) >= limit {
			break
		}

		// Check for next page link
		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break
		}

		links := ParseLinkHeader(linkHeader)
		nextURL, ok := links["next"]
		if !ok || nextURL == "" {
			break
		}

		// Treat the Link URL as opaque. If it's an absolute URL, use DoURL
		// directly. If it's a relative path, extract path+query for client.Do.
		parsed, err := url.Parse(nextURL)
		if err != nil {
			return allItems, meta, fmt.Errorf("failed to parse next URL: %w", err)
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

	meta.TotalItems = len(allItems)
	return allItems, meta, nil
}
