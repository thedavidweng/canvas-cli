package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ListDiscussions returns all discussion topics for a course.
// It sends GET /api/v1/courses/{courseID}/discussion_topics.
// The per_page parameter defaults to 100.
func ListDiscussions(ctx context.Context, client *Client, courseID string, query url.Values) ([]DiscussionTopic, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	var discussions []DiscussionTopic
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/discussion_topics", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &discussions,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list discussions for course %s: %w", courseID, err)
	}

	return discussions, meta.Pagination, nil
}

// GetDiscussion returns a single discussion topic by ID.
// It sends GET /api/v1/courses/{courseID}/discussion_topics/{discussionID}.
func GetDiscussion(ctx context.Context, client *Client, courseID, discussionID string) (DiscussionTopic, error) {
	var topic DiscussionTopic
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/discussion_topics/%s", courseID, discussionID),
		DecodeInto: &topic,
	})
	if err != nil {
		return topic, fmt.Errorf("get discussion %s in course %s: %w", discussionID, courseID, err)
	}

	return topic, nil
}

// ListDiscussionEntries returns all entries (replies) for a discussion topic.
// It sends GET /api/v1/courses/{courseID}/discussion_topics/{discussionID}/entries.
// The per_page parameter defaults to 100.
func ListDiscussionEntries(ctx context.Context, client *Client, courseID, discussionID string, query url.Values) ([]DiscussionEntry, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	var entries []DiscussionEntry
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/discussion_topics/%s/entries", courseID, discussionID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &entries,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list entries for discussion %s in course %s: %w", discussionID, courseID, err)
	}

	return entries, meta.Pagination, nil
}

// ReplyToDiscussion posts a new top-level entry to a discussion topic.
// It sends POST /api/v1/courses/{courseID}/discussion_topics/{discussionID}/entries.
func ReplyToDiscussion(ctx context.Context, client *Client, courseID, discussionID, message string) (DiscussionEntry, error) {
	var entry DiscussionEntry

	body, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return entry, fmt.Errorf("marshal reply request: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:    "POST",
		PathOrURL: fmt.Sprintf("/api/v1/courses/%s/discussion_topics/%s/entries", courseID, discussionID),
		Body:      bytes.NewReader(body),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		DecodeInto: &entry,
	})
	if err != nil {
		return entry, fmt.Errorf("reply to discussion %s in course %s: %w", discussionID, courseID, err)
	}

	return entry, nil
}

// CreateDiscussion creates a new discussion topic for a course.
func CreateDiscussion(ctx context.Context, client *Client, courseID, title, message string) (DiscussionTopic, error) {
	var topic DiscussionTopic
	payload := map[string]any{
		"title":   title,
		"message": message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return topic, fmt.Errorf("marshal discussion request: %w", err)
	}
	_, err = Request(ctx, client, RequestOptions{
		Method:     "POST",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/discussion_topics", courseID),
		Body:       bytes.NewReader(body),
		DecodeInto: &topic,
	})
	if err != nil {
		return topic, fmt.Errorf("create discussion in course %s: %w", courseID, err)
	}
	return topic, nil
}

// ReplyToEntry posts a reply to an existing discussion entry.
// It sends POST /api/v1/courses/{courseID}/discussion_topics/{discussionID}/entries/{entryID}/replies.
func ReplyToEntry(ctx context.Context, client *Client, courseID, discussionID, entryID, message string) (DiscussionEntry, error) {
	var entry DiscussionEntry

	body, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return entry, fmt.Errorf("marshal reply request: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:    "POST",
		PathOrURL: fmt.Sprintf("/api/v1/courses/%s/discussion_topics/%s/entries/%s/replies", courseID, discussionID, entryID),
		Body:      bytes.NewReader(body),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		DecodeInto: &entry,
	})
	if err != nil {
		return entry, fmt.Errorf("reply to entry %s in discussion %s in course %s: %w", entryID, discussionID, courseID, err)
	}

	return entry, nil
}
