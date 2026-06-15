package canvas

import (
	"fmt"
	"net/http"
	"testing"
)

func BenchmarkParseLinkHeader(b *testing.B) {
	header := `<https://school.instructure.com/api/v1/courses/1/topics?page=2>; rel="next", <https://school.instructure.com/api/v1/courses/1/topics?page=1>; rel="prev", <https://school.instructure.com/api/v1/courses/1/topics?page=1>; rel="first", <https://school.instructure.com/api/v1/courses/1/topics?page=5>; rel="last"`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseLinkHeader(header)
	}
}

func BenchmarkBackoffDelay(b *testing.B) {
	for attempt := 0; attempt < 5; attempt++ {
		b.Run(fmt.Sprintf("attempt-%d", attempt), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				backoffDelay(attempt)
			}
		})
	}
}

func BenchmarkCaptureRateMeta(b *testing.B) {
	resp := &http.Response{
		Header: http.Header{
			"X-Request-Cost":         []string{"1.5"},
			"X-Rate-Limit-Remaining": []string{"42.0"},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CaptureRateMeta(resp)
	}
}
