package canvas

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the HTTP client for the Canvas API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	userAgent  string
	version    string
	retries    int
}

// NewClient creates a new Canvas API client.
func NewClient(baseURL, token, version string, timeout time.Duration, retries int) *Client {
	trimmed := strings.TrimRight(baseURL, "/")
	if retries < 0 {
		retries = 0
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    trimmed,
		token:      token,
		userAgent:  "canvas-cli/" + version + " (+https://github.com/thedavidweng/canvas-cli)",
		version:    version,
		retries:    retries,
	}
}

// SetHTTPClient replaces the underlying HTTP client. This is useful for
// configuring custom redirect policies (e.g., for file uploads).
func (c *Client) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}

// Do executes an HTTP request with automatic retry for transient failures.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	return c.doWithRetry(ctx, method, path, query, body, c.retries)
}

// doWithRetry executes the request with retry logic for 429, 403-rate-limit, and 5xx.
func (c *Client) doWithRetry(ctx context.Context, method, path string, query url.Values, body io.Reader, maxRetries int) (*http.Response, error) {
	var lastResp *http.Response

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := c.doOnce(ctx, method, path, query, body)
		if err != nil {
			return nil, err
		}

		lastResp = resp

		retry, delay := ShouldRetry(resp, attempt, maxRetries)
		if !retry {
			return resp, nil
		}

		resp.Body.Close()

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return lastResp, nil
}

// doOnce executes a single HTTP request with default Canvas headers (no retry).
func (c *Client) doOnce(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	fullURL := c.baseURL + path
	if query != nil {
		fullURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json+canvas-string-ids")

	return c.httpClient.Do(req)
}

// DoURL executes a request to an absolute URL (for pagination next links
// and upload redirects). No base URL prepending.
func (c *Client) DoURL(ctx context.Context, method, absoluteURL string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, absoluteURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json+canvas-string-ids")

	return c.httpClient.Do(req)
}
