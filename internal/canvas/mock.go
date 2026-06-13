package canvas

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// MockCanvasAPI is a lightweight in-memory implementation of CanvasAPI for
// testing. Each method can be overridden by setting the corresponding Func
// field. When unset, methods return a 501 Not Implemented response (or
// no-op for SetHTTPClient).
type MockCanvasAPI struct {
	DoFunc             func(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error)
	DoWithHeadersFunc  func(ctx context.Context, method, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error)
	DoURLFunc          func(ctx context.Context, method, absoluteURL string, body io.Reader) (*http.Response, error)
	SetHTTPClientFunc  func(hc *http.Client)
}

// Do delegates to DoFunc, or returns 501 if unset.
func (m *MockCanvasAPI) Do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(ctx, method, path, query, body)
	}
	return notImplementedResponse(), nil
}

// DoWithHeaders delegates to DoWithHeadersFunc, or returns 501 if unset.
func (m *MockCanvasAPI) DoWithHeaders(ctx context.Context, method, path string, query url.Values, body io.Reader, headers http.Header) (*http.Response, error) {
	if m.DoWithHeadersFunc != nil {
		return m.DoWithHeadersFunc(ctx, method, path, query, body, headers)
	}
	return notImplementedResponse(), nil
}

// DoURL delegates to DoURLFunc, or returns 501 if unset.
func (m *MockCanvasAPI) DoURL(ctx context.Context, method, absoluteURL string, body io.Reader) (*http.Response, error) {
	if m.DoURLFunc != nil {
		return m.DoURLFunc(ctx, method, absoluteURL, body)
	}
	return notImplementedResponse(), nil
}

// SetHTTPClient delegates to SetHTTPClientFunc, or is a no-op if unset.
func (m *MockCanvasAPI) SetHTTPClient(hc *http.Client) {
	if m.SetHTTPClientFunc != nil {
		m.SetHTTPClientFunc(hc)
	}
}

// notImplementedResponse builds a 501 JSON response for unconfigured methods.
func notImplementedResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNotImplemented,
		Body:       io.NopCloser(strings.NewReader(`{"error":"method not configured in mock"}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}
