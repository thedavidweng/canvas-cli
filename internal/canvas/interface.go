package canvas

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// CanvasAPI is the subset of Client methods used by the canvas package
// functions (Request, Paginate, and direct callers like DownloadFile).
// Implementing this interface lets tests swap in a lightweight mock
// without standing up an HTTP server.
type CanvasAPI interface {
	// Do executes an HTTP request with automatic retry for transient failures.
	Do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error)

	// DoWithHeaders executes an HTTP request with custom headers and automatic retry.
	DoWithHeaders(ctx context.Context, method, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error)

	// DoURL executes a request to an absolute URL (for pagination next links
	// and upload redirects). No base URL prepending.
	DoURL(ctx context.Context, method, absoluteURL string, body io.Reader) (*http.Response, error)

	// SetHTTPClient replaces the underlying HTTP client.
	SetHTTPClient(hc *http.Client)
}

// Compile-time assertion: *Client must satisfy CanvasAPI.
var _ CanvasAPI = (*Client)(nil)
