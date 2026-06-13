package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ListAnnouncements returns all announcements for a course.
// It sends GET /api/v1/announcements with context_codes[]=course_{courseID}.
// The per_page parameter defaults to 100.
func ListAnnouncements(ctx context.Context, client *Client, courseID string, query url.Values) ([]DiscussionTopic, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	// Set the context_codes parameter required by the announcements endpoint
	query.Set("context_codes[]", fmt.Sprintf("course_%s", courseID))

	var announcements []DiscussionTopic
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/announcements",
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &announcements,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list announcements for course %s: %w", courseID, err)
	}

	return announcements, meta.Pagination, nil
}

// GetAnnouncement returns a single announcement by ID.
// Announcements are discussion topics, so this fetches from the discussion topics endpoint.
func GetAnnouncement(ctx context.Context, client *Client, courseID, announcementID string) (DiscussionTopic, error) {
	var topic DiscussionTopic
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/discussion_topics/%s", courseID, announcementID),
		DecodeInto: &topic,
	})
	if err != nil {
		return topic, fmt.Errorf("get announcement %s in course %s: %w", announcementID, courseID, err)
	}
	return topic, nil
}

// CreateAnnouncement creates a new announcement for a course.
// It sends POST /api/v1/courses/{courseID}/discussion_topics with is_announcement=true.
func CreateAnnouncement(ctx context.Context, client *Client, courseID, title, message string) (DiscussionTopic, error) {
	var topic DiscussionTopic

	payload := map[string]any{
		"is_announcement": true,
		"title":           title,
		"message":         message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return topic, fmt.Errorf("marshal announcement request: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:     "POST",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/discussion_topics", courseID),
		Body:       bytes.NewReader(body),
		DecodeInto: &topic,
	})
	if err != nil {
		return topic, fmt.Errorf("create announcement in course %s: %w", courseID, err)
	}

	return topic, nil
}
