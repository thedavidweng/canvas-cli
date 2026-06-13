package canvas

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestMockCanvasAPI_ImplementsInterface(t *testing.T) {
	var _ CanvasAPI = &MockCanvasAPI{}
}

func TestMockCanvasAPI_DoReturnsConfiguredResponse(t *testing.T) {
	course := Course{ID: "42", Name: "Intro to Go"}
	body, _ := json.Marshal(course)

	mock := &MockCanvasAPI{
		DoFunc: func(_ context.Context, method, path string, _ url.Values, _ io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}

	resp, err := mock.Do(context.Background(), "GET", "/api/v1/courses/42", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	var got Course
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "42" || got.Name != "Intro to Go" {
		t.Errorf("got %+v, want ID=42 Name=Intro to Go", got)
	}
}

func TestMockCanvasAPI_DoWithHeadersReturnsConfiguredResponse(t *testing.T) {
	mock := &MockCanvasAPI{
		DoWithHeadersFunc: func(_ context.Context, method, path string, _ url.Values, _ io.Reader, _ http.Header) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     http.Header{},
			}, nil
		},
	}

	resp, err := mock.DoWithHeaders(context.Background(), "GET", "/api/v1/courses", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
}

func TestMockCanvasAPI_DoURLReturnsConfiguredResponse(t *testing.T) {
	mock := &MockCanvasAPI{
		DoURLFunc: func(_ context.Context, method, absURL string, _ io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{},
			}, nil
		},
	}

	resp, err := mock.DoURL(context.Background(), "GET", "https://canvas.example.com/api/v1/next", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
}

func TestMockCanvasAPI_DoDefaultReturns501(t *testing.T) {
	mock := &MockCanvasAPI{}

	resp, err := mock.Do(context.Background(), "GET", "/api/v1/courses", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("got status %d, want 501", resp.StatusCode)
	}
}

func TestMockCanvasAPI_DoWithHeadersDefaultReturns501(t *testing.T) {
	mock := &MockCanvasAPI{}

	resp, err := mock.DoWithHeaders(context.Background(), "GET", "/api/v1/courses", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("got status %d, want 501", resp.StatusCode)
	}
}

func TestMockCanvasAPI_DoURLDefaultReturns501(t *testing.T) {
	mock := &MockCanvasAPI{}

	resp, err := mock.DoURL(context.Background(), "GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("got status %d, want 501", resp.StatusCode)
	}
}

func TestMockCanvasAPI_SetHTTPClient(t *testing.T) {
	mock := &MockCanvasAPI{}
	called := false
	mock.SetHTTPClientFunc = func(hc *http.Client) {
		called = true
	}

	mock.SetHTTPClient(&http.Client{})
	if !called {
		t.Error("SetHTTPClientFunc was not called")
	}
}

func TestMockCanvasAPI_SetHTTPClientDefaultNoOp(t *testing.T) {
	mock := &MockCanvasAPI{}
	// Should not panic.
	mock.SetHTTPClient(&http.Client{})
}
