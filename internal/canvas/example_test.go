package canvas

import (
	"fmt"
	"net/http"
)

func ExampleParseLinkHeader() {
	header := `<https://school.instructure.com/api/v1/courses?page=2>; rel="next", <https://school.instructure.com/api/v1/courses?page=1>; rel="prev"`
	links := ParseLinkHeader(header)
	fmt.Println(links["next"])
	fmt.Println(links["prev"])
	// Output:
	// https://school.instructure.com/api/v1/courses?page=2
	// https://school.instructure.com/api/v1/courses?page=1
}

func ExampleCaptureRateMeta() {
	resp := &http.Response{
		Header: http.Header{
			"X-Request-Cost":         []string{"2.0"},
			"X-Rate-Limit-Remaining": []string{"38.0"},
		},
	}
	meta := CaptureRateMeta(resp)
	fmt.Printf("cost=%.1f remaining=%.1f\n", meta.RequestCost, meta.Remaining)
	// Output:
	// cost=2.0 remaining=38.0
}
