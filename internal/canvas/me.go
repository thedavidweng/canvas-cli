package canvas

import (
	"context"
	"fmt"
)

// GetActivityStream returns the user's activity stream.
func GetActivityStream(ctx context.Context, client *Client) ([]ActivityItem, error) {
	var items []ActivityItem
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/users/self/activity_stream",
		DecodeInto: &items,
	})
	if err != nil {
		return nil, fmt.Errorf("get activity stream: %w", err)
	}
	return items, nil
}

// GetTodoItems returns the user's todo items.
func GetTodoItems(ctx context.Context, client *Client) ([]TodoItem, error) {
	var items []TodoItem
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/users/self/todo",
		DecodeInto: &items,
	})
	if err != nil {
		return nil, fmt.Errorf("get todo items: %w", err)
	}
	return items, nil
}

// GetUpcomingEvents returns the user's upcoming events.
func GetUpcomingEvents(ctx context.Context, client *Client) ([]UpcomingEvent, error) {
	var items []UpcomingEvent
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/users/self/upcoming_events",
		DecodeInto: &items,
	})
	if err != nil {
		return nil, fmt.Errorf("get upcoming events: %w", err)
	}
	return items, nil
}
