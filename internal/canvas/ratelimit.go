package canvas

import (
	"context"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// RateLimitMeta captures rate-limit headers from a Canvas response.
type RateLimitMeta struct {
	RequestCost float64 `json:"request_cost"`
	Remaining   float64 `json:"remaining"`
}

// CaptureRateMeta reads rate-limit headers from a response.
func CaptureRateMeta(resp *http.Response) *RateLimitMeta {
	meta := &RateLimitMeta{}

	if cost := resp.Header.Get("X-Request-Cost"); cost != "" {
		if v, err := strconv.ParseFloat(cost, 64); err == nil {
			meta.RequestCost = v
		}
	}

	if remaining := resp.Header.Get("X-Rate-Limit-Remaining"); remaining != "" {
		if v, err := strconv.ParseFloat(remaining, 64); err == nil {
			meta.Remaining = v
		}
	}

	return meta
}

// ShouldRetry determines if a request should be retried and calculates the backoff delay.
// It returns true if the response indicates a retryable condition and the attempt count
// hasn't exceeded maxRetries.
func ShouldRetry(resp *http.Response, attempt, maxRetries int) (bool, time.Duration) {
	if attempt >= maxRetries {
		return false, 0
	}

	// Check Retry-After header first
	var retryAfterDuration time.Duration
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if seconds, err := strconv.Atoi(ra); err == nil {
			retryAfterDuration = time.Duration(seconds) * time.Second
		}
	}

	switch {
	case resp.StatusCode == 429:
		// Always retry 429
		if retryAfterDuration > 0 {
			return true, retryAfterDuration
		}
		return true, backoffDelay(attempt)

	case resp.StatusCode == 403:
		// Retry 403 only when rate limit is exhausted
		if remaining := resp.Header.Get("X-Rate-Limit-Remaining"); remaining == "0" {
			if retryAfterDuration > 0 {
				return true, retryAfterDuration
			}
			return true, backoffDelay(attempt)
		}
		return false, 0

	case resp.StatusCode >= 500 && resp.StatusCode <= 599:
		// Retry transient 5xx
		if retryAfterDuration > 0 {
			return true, retryAfterDuration
		}
		return true, backoffDelay(attempt)

	default:
		return false, 0
	}
}

// backoffDelay calculates a bounded exponential backoff with jitter.
// Base delay is 1 second, max is 30 seconds.
func backoffDelay(attempt int) time.Duration {
	base := float64(time.Second)
	maxDelay := float64(30 * time.Second)

	delay := base * math.Pow(2, float64(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter: random value between 0 and 25% of the delay
	jitter := rand.Float64() * delay * 0.25
	return time.Duration(delay + jitter)
}

// DoWithRetry executes an HTTP request with automatic retry for transient failures.
// This is the standalone retry function; Client.Do now has built-in retry.
func DoWithRetry(ctx context.Context, client *Client, method, path string, query url.Values, body io.Reader, maxRetries int) (*http.Response, error) {
	var lastResp *http.Response

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := client.doOnce(ctx, method, path, query, body)
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
