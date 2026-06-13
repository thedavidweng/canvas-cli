package canvas

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCaptureRateMeta(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"X-Request-Cost":         {"1.5"},
			"X-Rate-Limit-Remaining": {"998.5"},
		},
	}
	meta := CaptureRateMeta(resp)

	if meta == nil {
		t.Fatal("CaptureRateMeta returned nil")
	}
	if meta.RequestCost != 1.5 {
		t.Errorf("RequestCost = %f, want 1.5", meta.RequestCost)
	}
	if meta.Remaining != 998.5 {
		t.Errorf("Remaining = %f, want 998.5", meta.Remaining)
	}
}

func TestCaptureRateMetaMissingHeaders(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	meta := CaptureRateMeta(resp)

	if meta == nil {
		t.Fatal("CaptureRateMeta returned nil")
	}
	if meta.RequestCost != 0 {
		t.Errorf("RequestCost = %f, want 0", meta.RequestCost)
	}
	if meta.Remaining != 0 {
		t.Errorf("Remaining = %f, want 0", meta.Remaining)
	}
}

func TestShouldRetry429(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	retry, delay := ShouldRetry(resp, 0, 3)

	if !retry {
		t.Error("ShouldRetry should return true for 429")
	}
	if delay <= 0 {
		t.Errorf("delay should be positive, got %v", delay)
	}
}

func TestShouldRetryRespectsRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": {"5"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	retry, delay := ShouldRetry(resp, 0, 3)

	if !retry {
		t.Error("ShouldRetry should return true for 429 with Retry-After")
	}
	if delay < 5*time.Second {
		t.Errorf("delay = %v, want >= 5s", delay)
	}
}

func TestShouldRetryTransient5xx(t *testing.T) {
	codes := []int{500, 502, 503, 504}
	for _, code := range codes {
		resp := &http.Response{
			StatusCode: code,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("")),
		}
		retry, _ := ShouldRetry(resp, 0, 3)
		if !retry {
			t.Errorf("ShouldRetry should return true for %d", code)
		}
	}
}

func TestShouldRetry403WithRateLimitExhausted(t *testing.T) {
	resp := &http.Response{
		StatusCode: 403,
		Header:     http.Header{"X-Rate-Limit-Remaining": {"0"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	retry, _ := ShouldRetry(resp, 0, 3)

	if !retry {
		t.Error("ShouldRetry should return true for 403 with rate limit exhausted")
	}
}

func TestShouldRetry403WithoutRateLimitHeader(t *testing.T) {
	resp := &http.Response{
		StatusCode: 403,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	retry, _ := ShouldRetry(resp, 0, 3)

	if retry {
		t.Error("ShouldRetry should return false for normal 403")
	}
}

func TestShouldNotRetryNormal4xx(t *testing.T) {
	codes := []int{400, 401, 404, 405, 409, 422}
	for _, code := range codes {
		resp := &http.Response{
			StatusCode: code,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("")),
		}
		retry, _ := ShouldRetry(resp, 0, 3)
		if retry {
			t.Errorf("ShouldRetry should return false for %d", code)
		}
	}
}

func TestShouldNotRetryMaxRetriesExhausted(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	retry, _ := ShouldRetry(resp, 3, 3)

	if retry {
		t.Error("ShouldRetry should return false when max retries exhausted")
	}
}

func TestShouldRetryBackoffIncreases(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	var delays []time.Duration
	for attempt := 0; attempt < 3; attempt++ {
		_, delay := ShouldRetry(resp, attempt, 3)
		delays = append(delays, delay)
	}

	// Verify delays are generally increasing (accounting for jitter)
	if delays[0] >= delays[2] {
		t.Errorf("backoff should increase: delays = %v", delays)
	}
}

func TestDoWithRetryRetries429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	resp, err := DoWithRetry(context.Background(), c, "GET", "/api/v1/test", nil, nil, 3)
	if err != nil {
		t.Fatalf("DoWithRetry() error: %v", err)
	}
	resp.Body.Close()

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestDoWithRetryExhaustsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	resp, err := DoWithRetry(context.Background(), c, "GET", "/api/v1/test", nil, nil, 2)
	if err != nil {
		t.Fatalf("DoWithRetry() error: %v", err)
	}
	defer resp.Body.Close()

	// Should return the last 429 response
	if resp.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", resp.StatusCode)
	}
}

func TestDoWithRetryDoesNotRetrySuccess(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	resp, err := DoWithRetry(context.Background(), c, "GET", "/api/v1/test", nil, nil, 3)
	if err != nil {
		t.Fatalf("DoWithRetry() error: %v", err)
	}
	resp.Body.Close()

	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestDoWithRetryContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.1.0", 5*time.Second, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := DoWithRetry(ctx, c, "GET", "/api/v1/test", nil, nil, 10)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
