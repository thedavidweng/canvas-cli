package canvas

import "testing"

func FuzzParseLinkHeader(f *testing.F) {
	f.Add(`<https://example.com/api/v1/courses?page=2>; rel="next"`)
	f.Add(`<https://a.com>; rel="next", <https://b.com>; rel="prev"`)
	f.Add(``)
	f.Add(`garbage input`)
	f.Add(`<https://x.com>; rel=next`)
	f.Fuzz(func(t *testing.T, input string) {
		links := ParseLinkHeader(input)
		if links == nil {
			t.Fatal("ParseLinkHeader returned nil map")
		}
	})
}
