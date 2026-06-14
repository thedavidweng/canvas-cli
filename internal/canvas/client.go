package canvas

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is the HTTP client for the Canvas API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	cookie     string
	csrfToken  string
	csrfCached string
	csrfMu     sync.Mutex // guards csrfCached
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

// WithCookie sets cookie-based authentication on the client. If token is also
// set (via NewClient), token auth takes precedence. csrfToken may be empty;
// the client will attempt to cache it from response headers.
func (c *Client) WithCookie(cookie, csrfToken string) *Client {
	c.cookie = cookie
	c.csrfToken = csrfToken
	return c
}

// SetHTTPClient replaces the underlying HTTP client. This is useful for
// configuring custom redirect policies (e.g., for file uploads).
func (c *Client) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}

// Do executes an HTTP request with automatic retry for transient failures.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	return c.DoWithHeaders(ctx, method, path, query, body, nil)
}

// DoWithHeaders executes an HTTP request with custom headers and automatic retry.
func (c *Client) DoWithHeaders(ctx context.Context, method, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error) {
	return c.doWithRetry(ctx, method, path, query, body, c.retries, headers)
}

// doWithRetry executes the request with retry logic for 429, 403-rate-limit, and 5xx.
func (c *Client) doWithRetry(ctx context.Context, method, path string, query url.Values, body io.Reader, maxRetries int, headers http.Header) (*http.Response, error) {
	var lastResp *http.Response

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := c.doOnce(ctx, method, path, query, body, headers)
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
func (c *Client) doOnce(ctx context.Context, method, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error) {
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

	// Auth header: token takes precedence over cookie.
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json+canvas-string-ids")

	// CSRF header for unsafe methods when using cookie auth.
	if c.cookie != "" && c.token == "" && isUnsafeMethod(method) {
		csrf := c.csrfToken
		if csrf == "" {
			c.csrfMu.Lock()
			csrf = c.csrfCached
			c.csrfMu.Unlock()
		}
		if csrf == "" {
			return nil, fmt.Errorf("CSRF token required for mutation with cookie auth")
		}
		req.Header.Set("X-CSRF-Token", csrf)
	}

	// Apply custom headers (override defaults if needed).
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}

	// Use per-request redirect policy for cookie auth to avoid mutating the shared client.
	hc := c.httpClient
	if c.cookie != "" && c.token == "" {
		hc = &http.Client{
			Transport: c.httpClient.Transport,
			CheckRedirect: func(redirectReq *http.Request, via []*http.Request) error {
				if isAuthRedirect(redirectReq.URL.String()) {
					return &CookieSessionExpiredError{Location: redirectReq.URL.String()}
				}
				if len(via) > 0 && isUnsafeMethod(via[0].Method) {
					return http.ErrUseLastResponse
				}
				redirectReq.Header.Del("Cookie")
				redirectReq.Header.Del("Authorization")
				redirectReq.Header.Del("X-CSRF-Token")
				return nil
			},
			Timeout: c.httpClient.Timeout,
		}
	}

	resp, err := hc.Do(req)
	if err != nil {
		var csErr *CookieSessionExpiredError
		if errors.As(err, &csErr) {
			return nil, csErr
		}
		return nil, err
	}

	// Cache X-CSRF-Token from response header.
	if csrf := resp.Header.Get("X-CSRF-Token"); csrf != "" {
		c.csrfMu.Lock()
		c.csrfCached = csrf
		c.csrfMu.Unlock()
	}

	return resp, nil
}

// DoURL executes a request to an absolute URL (for pagination next links
// and upload redirects). No base URL prepending.
// Auth headers are only sent when the URL host matches the configured base URL host.
func (c *Client) DoURL(ctx context.Context, method, absoluteURL string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, absoluteURL, body)
	if err != nil {
		return nil, err
	}

	// Only send auth headers to the same host as the configured base URL.
	sameHost := urlHostMatches(c.baseURL, absoluteURL)
	if sameHost {
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else if c.cookie != "" {
			req.Header.Set("Cookie", c.cookie)
		}
		if c.cookie != "" && c.token == "" && isUnsafeMethod(method) {
			csrf := c.csrfToken
			if csrf == "" {
				c.csrfMu.Lock()
				csrf = c.csrfCached
				c.csrfMu.Unlock()
			}
			if csrf == "" {
				return nil, fmt.Errorf("CSRF token required for mutation with cookie auth")
			}
			req.Header.Set("X-CSRF-Token", csrf)
		}
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json+canvas-string-ids")

	// Per-request redirect policy for cookie auth.
	hc := c.httpClient
	if c.cookie != "" && c.token == "" {
		hc = &http.Client{
			Transport: c.httpClient.Transport,
			CheckRedirect: func(redirectReq *http.Request, via []*http.Request) error {
				if isAuthRedirect(redirectReq.URL.String()) {
					return &CookieSessionExpiredError{Location: redirectReq.URL.String()}
				}
				if len(via) > 0 && isUnsafeMethod(via[0].Method) {
					return http.ErrUseLastResponse
				}
				redirectReq.Header.Del("Cookie")
				redirectReq.Header.Del("Authorization")
				redirectReq.Header.Del("X-CSRF-Token")
				return nil
			},
			Timeout: c.httpClient.Timeout,
		}
	}

	resp, err := hc.Do(req)
	if err != nil {
		var csErr *CookieSessionExpiredError
		if errors.As(err, &csErr) {
			return nil, csErr
		}
		return nil, err
	}

	if csrf := resp.Header.Get("X-CSRF-Token"); csrf != "" {
		c.csrfMu.Lock()
		c.csrfCached = csrf
		c.csrfMu.Unlock()
	}

	return resp, nil
}

// isUnsafeMethod returns true for HTTP methods that can modify state.
func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// urlHostMatches returns true if two absolute URLs share the same host.
func urlHostMatches(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return strings.EqualFold(ua.Hostname(), ub.Hostname())
}
