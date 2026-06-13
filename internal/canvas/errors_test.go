package canvas

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func newResponse(status int, body string, headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNormalizeError401(t *testing.T) {
	resp := newResponse(401, `{"message":"Unauthorized"}`, nil)
	env := NormalizeError(resp, "courses.list")

	if env.OK {
		t.Error("OK should be false")
	}
	if env.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if env.Error.Code != "CANVAS_AUTH_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_AUTH_ERROR")
	}
	if env.Error.Category != "auth" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "auth")
	}
	if env.Error.Status != 401 {
		t.Errorf("Status = %d, want 401", env.Error.Status)
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for auth errors")
	}
	if env.Meta.Command != "courses.list" {
		t.Errorf("Command = %q, want %q", env.Meta.Command, "courses.list")
	}
}

func TestNormalizeError403(t *testing.T) {
	resp := newResponse(403, `{"message":"Forbidden"}`, nil)
	env := NormalizeError(resp, "courses.update")

	if env.Error.Code != "CANVAS_PERMISSION_DENIED" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_PERMISSION_DENIED")
	}
	if env.Error.Category != "permission" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "permission")
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for permission errors")
	}
}

func TestNormalizeError404(t *testing.T) {
	resp := newResponse(404, `{"message":"Not Found"}`, nil)
	env := NormalizeError(resp, "courses.get")

	if env.Error.Code != "CANVAS_NOT_FOUND" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_NOT_FOUND")
	}
	if env.Error.Category != "not_found" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "not_found")
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for not found errors")
	}
}

func TestNormalizeError422(t *testing.T) {
	resp := newResponse(422, `{"message":"Unprocessable Entity","errors":[{"message":"Name is required"}]}`, nil)
	env := NormalizeError(resp, "assignments.create")

	if env.Error.Code != "CANVAS_VALIDATION_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_VALIDATION_ERROR")
	}
	if env.Error.Category != "validation" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "validation")
	}
	if env.Error.Retryable {
		t.Error("Retryable should be false for validation errors")
	}
}

func TestNormalizeErrorCanvasRequestID(t *testing.T) {
	resp := newResponse(400, `{"message":"Bad Request"}`, map[string]string{
		"X-Request-Id": "req-abc-123",
	})
	env := NormalizeError(resp, "api.get")

	if env.Error.CanvasRequestID != "req-abc-123" {
		t.Errorf("CanvasRequestID = %q, want %q", env.Error.CanvasRequestID, "req-abc-123")
	}
}

func TestNormalizeErrorBodyPreserved(t *testing.T) {
	body := `{"message":"Bad Request","errors":[{"message":"invalid field"}]}`
	resp := newResponse(400, body, nil)
	env := NormalizeError(resp, "api.get")

	if env.Error.ResponseBody == nil {
		t.Fatal("ResponseBody should not be nil")
	}
	bodyMap, ok := env.Error.ResponseBody.(map[string]any)
	if !ok {
		t.Fatalf("ResponseBody should be a map, got %T", env.Error.ResponseBody)
	}
	if bodyMap["message"] != "Bad Request" {
		t.Errorf("ResponseBody[message] = %v, want %q", bodyMap["message"], "Bad Request")
	}
}

func TestNormalizeError500(t *testing.T) {
	resp := newResponse(500, `{"message":"Internal Server Error"}`, nil)
	env := NormalizeError(resp, "api.get")

	if env.Error.Code != "CANVAS_SERVER_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_SERVER_ERROR")
	}
	if env.Error.Category != "server" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "server")
	}
	if !env.Error.Retryable {
		t.Error("Retryable should be true for 5xx errors")
	}
}

func TestNormalizeErrorGenericCode(t *testing.T) {
	resp := newResponse(409, `{"message":"Conflict"}`, nil)
	env := NormalizeError(resp, "api.get")

	if env.Error.Code != "CANVAS_API_ERROR" {
		t.Errorf("Code = %q, want %q", env.Error.Code, "CANVAS_API_ERROR")
	}
	if env.Error.Category != "api" {
		t.Errorf("Category = %q, want %q", env.Error.Category, "api")
	}
}
