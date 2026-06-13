package canvas

import (
	"encoding/json"
	"io"
	"net/http"
)

// NormalizeError converts an HTTP error response into a structured Envelope.
func NormalizeError(resp *http.Response, command string) Envelope {
	env := Envelope{
		OK: false,
		Meta: Meta{
			SchemaVersion: SchemaVersion,
			Command:       command,
		},
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	var bodyMap map[string]any
	if err == nil && len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &bodyMap)
	}

	errInfo := &ErrorInfo{
		Status:       resp.StatusCode,
		ResponseBody: bodyMap,
	}

	// Extract message from body if available, with fallback.
	if bodyMap != nil {
		if msg, ok := bodyMap["message"].(string); ok {
			errInfo.Message = msg
		}
	}
	if errInfo.Message == "" {
		errInfo.Message = http.StatusText(resp.StatusCode)
	}

	// Map status codes to error codes, categories, and retryable flag.
	switch resp.StatusCode {
	case http.StatusUnauthorized: // 401
		errInfo.Code = "CANVAS_AUTH_ERROR"
		errInfo.Category = "auth"
	case http.StatusForbidden: // 403
		errInfo.Code = "CANVAS_PERMISSION_DENIED"
		errInfo.Category = "permission"
		// 403 with rate limit exhausted is retryable.
		if resp.Header.Get("X-Rate-Limit-Remaining") == "0" {
			errInfo.Code = "CANVAS_RATE_LIMIT"
			errInfo.Category = "rate_limit"
			errInfo.Retryable = true
		}
	case http.StatusNotFound: // 404
		errInfo.Code = "CANVAS_NOT_FOUND"
		errInfo.Category = "not_found"
	case http.StatusUnprocessableEntity: // 422
		errInfo.Code = "CANVAS_VALIDATION_ERROR"
		errInfo.Category = "validation"
	case http.StatusTooManyRequests: // 429
		errInfo.Code = "CANVAS_RATE_LIMIT"
		errInfo.Category = "rate_limit"
		errInfo.Retryable = true
	default:
		if resp.StatusCode >= 500 {
			errInfo.Code = "CANVAS_SERVER_ERROR"
			errInfo.Category = "server"
			errInfo.Retryable = true
		} else {
			errInfo.Code = "CANVAS_API_ERROR"
			errInfo.Category = "api"
		}
	}

	if reqID := resp.Header.Get("X-Request-Id"); reqID != "" {
		errInfo.CanvasRequestID = reqID
	}

	env.Error = errInfo
	return env
}
