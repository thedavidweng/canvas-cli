// Package output provides stable JSON envelope construction, JSON serialization,
// and human-readable table rendering for canvas-cli commands.
package output

import "github.com/thedavidweng/canvas-cli/internal/canvas"

// SchemaVersion is re-exported from the canvas package for convenience.
const SchemaVersion = canvas.SchemaVersion

// NewSuccess builds a success Envelope with ok=true and the given data.
// Optional meta overrides are merged into the base Meta (command and schema_version).
func NewSuccess(data any, command string, metaOverrides ...canvas.Meta) canvas.Envelope {
	m := canvas.Meta{
		SchemaVersion: SchemaVersion,
		Command:       command,
	}
	if len(metaOverrides) > 0 {
		m = mergeMeta(m, metaOverrides[0])
	}
	return canvas.Envelope{
		OK:   true,
		Data: data,
		Meta: m,
	}
}

// NewError builds an error Envelope with ok=false.
// Optional meta overrides are merged into the base Meta.
func NewError(err canvas.ErrorInfo, command string, metaOverrides ...canvas.Meta) canvas.Envelope {
	m := canvas.Meta{
		SchemaVersion: SchemaVersion,
		Command:       command,
	}
	if len(metaOverrides) > 0 {
		m = mergeMeta(m, metaOverrides[0])
	}
	return canvas.Envelope{
		OK:    false,
		Error: &err,
		Meta:  m,
	}
}

// mergeMeta applies non-zero fields from override onto base.
func mergeMeta(base, override canvas.Meta) canvas.Meta {
	if override.Profile != "" {
		base.Profile = override.Profile
	}
	if override.BaseURL != "" {
		base.BaseURL = override.BaseURL
	}
	if override.DurationMS != 0 {
		base.DurationMS = override.DurationMS
	}
	if override.RequestCount != 0 {
		base.RequestCount = override.RequestCount
	}
	if override.Paginated {
		base.Paginated = override.Paginated
	}
	if override.PageSize != 0 {
		base.PageSize = override.PageSize
	}
	if override.Limit != nil {
		base.Limit = override.Limit
	}
	if override.RateLimit != nil {
		base.RateLimit = override.RateLimit
	}
	if len(override.Warnings) > 0 {
		base.Warnings = override.Warnings
	}
	return base
}
