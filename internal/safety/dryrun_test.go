package safety

import (
	"strings"
	"testing"
)

func TestFormatPreview_BasicFields(t *testing.T) {
	p := Preview{
		Method:         "PUT",
		Path:           "/api/v1/courses/123/assignments/456/submissions/789",
		ResourceIDs:    []string{"123", "456", "789"},
		PayloadSummary: `{"grade": "95"}`,
	}
	out := FormatPreview(p)

	if !strings.Contains(out, "PUT") {
		t.Error("output should contain method PUT")
	}
	if !strings.Contains(out, "/api/v1/courses/123/assignments/456/submissions/789") {
		t.Error("output should contain path")
	}
	if !strings.Contains(out, "123") {
		t.Error("output should contain resource ID 123")
	}
	if !strings.Contains(out, "456") {
		t.Error("output should contain resource ID 456")
	}
	if !strings.Contains(out, "789") {
		t.Error("output should contain resource ID 789")
	}
	if !strings.Contains(out, "grade") {
		t.Error("output should contain payload summary content")
	}
}

func TestFormatPreview_EmptyResourceIDs(t *testing.T) {
	p := Preview{
		Method:         "POST",
		Path:           "/api/v1/courses",
		ResourceIDs:    []string{},
		PayloadSummary: `{"name": "New Course"}`,
	}
	out := FormatPreview(p)

	if !strings.Contains(out, "POST") {
		t.Error("output should contain method")
	}
	if !strings.Contains(out, "/api/v1/courses") {
		t.Error("output should contain path")
	}
	if !strings.Contains(out, "New Course") {
		t.Error("output should contain payload content")
	}
}

func TestFormatPreview_NilResourceIDs(t *testing.T) {
	p := Preview{
		Method:         "DELETE",
		Path:           "/api/v1/courses/1/files/2",
		ResourceIDs:    nil,
		PayloadSummary: "",
	}
	out := FormatPreview(p)

	if !strings.Contains(out, "DELETE") {
		t.Error("output should contain method")
	}
	if !strings.Contains(out, "/api/v1/courses/1/files/2") {
		t.Error("output should contain path")
	}
}

func TestFormatPreview_LongPayloadSummary(t *testing.T) {
	// PayloadSummary should be truncated or summarized, not dumped in full
	longPayload := strings.Repeat("x", 500)
	p := Preview{
		Method:         "PUT",
		Path:           "/api/v1/courses/1/assignments/2",
		ResourceIDs:    []string{"1", "2"},
		PayloadSummary: longPayload,
	}
	out := FormatPreview(p)

	// The output should be reasonable length, not blowing up
	if len(out) > 1000 {
		t.Errorf("output too long (%d chars), expected summarization", len(out))
	}
}

func TestFormatPreview_MultipleResourceIDs(t *testing.T) {
	p := Preview{
		Method:         "PUT",
		Path:           "/api/v1/courses/10/assignments/20/submissions/30",
		ResourceIDs:    []string{"10", "20", "30"},
		PayloadSummary: `{"score": 88}`,
	}
	out := FormatPreview(p)

	for _, id := range []string{"10", "20", "30"} {
		if !strings.Contains(out, id) {
			t.Errorf("output should contain resource ID %s", id)
		}
	}
}
