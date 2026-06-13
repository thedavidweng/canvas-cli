package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// UpdateAssignment updates an assignment with the given fields.
// It sends PUT /api/v1/courses/{courseID}/assignments/{assignmentID}
// with the updates map wrapped in an assignment key.
func UpdateAssignment(ctx context.Context, client *Client, courseID, assignmentID string, updates map[string]any) (Assignment, error) {
	var a Assignment

	payload := map[string]any{
		"assignment": updates,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return a, fmt.Errorf("marshal assignment update: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:     "PUT",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/assignments/%s", courseID, assignmentID),
		Body:       bytes.NewReader(body),
		DecodeInto: &a,
	})
	if err != nil {
		return a, fmt.Errorf("update assignment %s in course %s: %w", assignmentID, courseID, err)
	}

	return a, nil
}

// UpdatePage updates a wiki page with the given fields.
// It sends PUT /api/v1/courses/{courseID}/pages/{pageURL}
// with the updates map wrapped in a wiki_page key.
func UpdatePage(ctx context.Context, client *Client, courseID, pageURL string, updates map[string]any) (Page, error) {
	var page Page

	payload := map[string]any{
		"wiki_page": updates,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return page, fmt.Errorf("marshal page update: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:     "PUT",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/pages/%s", courseID, pageURL),
		Body:       bytes.NewReader(body),
		DecodeInto: &page,
	})
	if err != nil {
		return page, fmt.Errorf("update page %s in course %s: %w", pageURL, courseID, err)
	}

	return page, nil
}

// PublishModule publishes or unpublishes a module.
// It sends PUT /api/v1/courses/{courseID}/modules/{moduleID}
// with module.published set to the given value.
func PublishModule(ctx context.Context, client *Client, courseID, moduleID string, published bool) (Module, error) {
	var mod Module

	payload := map[string]any{
		"module": map[string]any{
			"published": published,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return mod, fmt.Errorf("marshal module publish request: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:     "PUT",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/modules/%s", courseID, moduleID),
		Body:       bytes.NewReader(body),
		DecodeInto: &mod,
	})
	if err != nil {
		return mod, fmt.Errorf("publish module %s in course %s: %w", moduleID, courseID, err)
	}

	return mod, nil
}
