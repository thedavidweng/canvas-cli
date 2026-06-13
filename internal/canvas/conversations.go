package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// ListConversations returns all conversations for the authenticated user.
// It sends GET /api/v1/conversations with pagination.
// The per_page parameter defaults to 100.
func ListConversations(ctx context.Context, client *Client, opts RequestOptions) ([]Conversation, PaginationMeta, error) {
	var conversations []Conversation
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  "/api/v1/conversations",
		Paginate:   true,
		PageSize:   opts.PageSize,
		DecodeInto: &conversations,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list conversations: %w", err)
	}

	return conversations, meta.Pagination, nil
}

// GetConversation returns a single conversation by ID.
// It sends GET /api/v1/conversations/{conversationID}.
func GetConversation(ctx context.Context, client *Client, conversationID string) (Conversation, error) {
	var conversation Conversation
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/conversations/%s", conversationID),
		DecodeInto: &conversation,
	})
	if err != nil {
		return conversation, fmt.Errorf("get conversation %s: %w", conversationID, err)
	}

	return conversation, nil
}

// SendMessage creates a new conversation with the given recipients, subject, and body.
// It sends POST /api/v1/conversations.
func SendMessage(ctx context.Context, client *Client, recipients []string, subject, body string) (Conversation, error) {
	payload := map[string]any{
		"recipients": recipients,
		"subject":    subject,
		"body":       body,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Conversation{}, fmt.Errorf("marshal send message payload: %w", err)
	}

	var conversation Conversation
	_, err = Request(ctx, client, RequestOptions{
		Method:     "POST",
		PathOrURL:  "/api/v1/conversations",
		Body:       bytes.NewReader(payloadBytes),
		DecodeInto: &conversation,
	})
	if err != nil {
		return Conversation{}, fmt.Errorf("send message: %w", err)
	}

	return conversation, nil
}

// ReplyToConversation adds a message to an existing conversation.
// It sends POST /api/v1/conversations/{conversationID}/add_message.
func ReplyToConversation(ctx context.Context, client *Client, conversationID, body string) (Conversation, error) {
	payload := map[string]any{
		"body": body,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Conversation{}, fmt.Errorf("marshal reply payload: %w", err)
	}

	var conversation Conversation
	_, err = Request(ctx, client, RequestOptions{
		Method:     "POST",
		PathOrURL:  fmt.Sprintf("/api/v1/conversations/%s/add_message", conversationID),
		Body:       bytes.NewReader(payloadBytes),
		DecodeInto: &conversation,
	})
	if err != nil {
		return Conversation{}, fmt.Errorf("reply to conversation %s: %w", conversationID, err)
	}

	return conversation, nil
}

// ArchiveConversation archives a conversation by setting its workflow_state to "archived".
// It sends PUT /api/v1/conversations/{conversationID}.
func ArchiveConversation(ctx context.Context, client *Client, conversationID string) error {
	payload := map[string]any{
		"workflow_state": "archived",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal archive payload: %w", err)
	}

	_, err = Request(ctx, client, RequestOptions{
		Method:    "PUT",
		PathOrURL: fmt.Sprintf("/api/v1/conversations/%s", conversationID),
		Body:      bytes.NewReader(payloadBytes),
	})
	if err != nil {
		return fmt.Errorf("archive conversation %s: %w", conversationID, err)
	}

	return nil
}
