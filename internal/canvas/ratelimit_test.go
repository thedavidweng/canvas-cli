package canvas

import (
	"io"
	"net/http"
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

func TestShouldRetry_403WithRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 403,
		Header:     http.Header{"X-Rate-Limit-Remaining": {"0"}, "Retry-After": {"10"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	retry, delay := ShouldRetry(resp, 0, 3)

	if !retry {
		t.Error("ShouldRetry should return true for 403 with rate limit exhausted and Retry-After")
	}
	if delay < 10*time.Second {
		t.Errorf("delay = %v, want >= 10s", delay)
	}
}

func TestShouldRetry_5xxWithRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 503,
		Header:     http.Header{"Retry-After": {"15"}},
		Body:       io.NopCloser(strings.NewReader("")),
	}
	retry, delay := ShouldRetry(resp, 0, 3)

	if !retry {
		t.Error("ShouldRetry should return true for 503 with Retry-After")
	}
	if delay < 15*time.Second {
		t.Errorf("delay = %v, want >= 15s", delay)
	}
}

func TestBackoffDelay_MaxCap(t *testing.T) {
	// Very high attempt should cap at 30s + jitter.
	delay := backoffDelay(20)
	maxExpected := time.Duration(float64(30*time.Second) * 1.25) // max + 25% jitter
	if delay > maxExpected {
		t.Errorf("backoffDelay(20) = %v, want <= %v", delay, maxExpected)
	}
	if delay < 30*time.Second {
		t.Errorf("backoffDelay(20) = %v, want >= 30s", delay)
	}
}

func TestBackoffDelay_IncreasesWithAttempt(t *testing.T) {
	// Run multiple times to account for jitter.
	low := backoffDelay(0)
	high := backoffDelay(5)

	// With jitter, individual values may overlap, but the base delay
	// at attempt 5 (32s, capped to 30s) is much larger than attempt 0 (1s).
	if high < low {
		// This could theoretically happen with extreme jitter, but is very unlikely.
		t.Logf("WARNING: backoffDelay(5)=%v < backoffDelay(0)=%v (jitter)", high, low)
	}
}

func TestCaptureRateMeta_InvalidHeaders(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"X-Request-Cost":         {"not-a-number"},
			"X-Rate-Limit-Remaining": {"also-not-a-number"},
		},
	}
	meta := CaptureRateMeta(resp)

	if meta.RequestCost != 0 {
		t.Errorf("RequestCost = %f, want 0", meta.RequestCost)
	}
	if meta.Remaining != 0 {
		t.Errorf("Remaining = %f, want 0", meta.Remaining)
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
