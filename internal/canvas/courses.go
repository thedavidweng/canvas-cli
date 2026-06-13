package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// ListCourses returns all courses for the authenticated user.
// It sends GET /api/v1/courses with the given query parameters.
// The per_page parameter defaults to 100.
func ListCourses(ctx context.Context, client *Client, query url.Values) ([]Course, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	var courses []Course
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/courses",
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &courses,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list courses: %w", err)
	}

	return courses, meta.Pagination, nil
}

// GetCourse returns a single course by ID.
// It sends GET /api/v1/courses/{courseID} with include[]=term by default.
func GetCourse(ctx context.Context, client *Client, courseID string, query url.Values) (Course, error) {
	if query == nil {
		query = url.Values{}
	}

	// Always include term unless caller already has include[] set
	includes := query["include[]"]
	hasTerm := false
	for _, inc := range includes {
		if inc == "term" {
			hasTerm = true
			break
		}
	}
	if !hasTerm {
		query.Add("include[]", "term")
	}

	var course Course
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s", courseID),
		Query:      query,
		DecodeInto: &course,
	})
	if err != nil {
		return course, fmt.Errorf("get course %s: %w", courseID, err)
	}

	return course, nil
}

// ListCourseTabs returns the navigation tabs for a course.
// It sends GET /api/v1/courses/{courseID}/tabs.
func ListCourseTabs(ctx context.Context, client *Client, courseID string) ([]Tab, error) {
	var tabs []Tab
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/tabs", courseID),
		DecodeInto: &tabs,
	})
	if err != nil {
		return nil, fmt.Errorf("list course tabs %s: %w", courseID, err)
	}

	return tabs, nil
}
