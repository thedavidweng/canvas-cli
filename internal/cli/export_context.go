package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// allExportSections lists every section the export-context command can fetch.
var allExportSections = []string{
	"course", "tabs", "modules", "assignments", "assignment_groups",
	"files", "pages", "announcements", "discussions", "submissions", "grades",
}

// ExportContextOpts controls which sections to fetch and filtering behavior.
type ExportContextOpts struct {
	Include []string
	Since   string
}

// ExportMeta holds metadata about the export run.
type ExportMeta struct {
	GeneratedAt       string   `json:"generated_at"`
	CourseID          string   `json:"course_id"`
	SectionsRequested []string `json:"sections_requested"`
	SectionsSucceeded []string `json:"sections_succeeded"`
	SectionsFailed    []string `json:"sections_failed,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	RequestCount      int      `json:"request_count"`
	DurationMS        int64    `json:"duration_ms"`
}

// ExportResult is the top-level JSON shape for the export.
type ExportResult struct {
	Course           any        `json:"course,omitempty"`
	Tabs             []any      `json:"tabs,omitempty"`
	Modules          []any      `json:"modules,omitempty"`
	Assignments      []any      `json:"assignments,omitempty"`
	AssignmentGroups []any      `json:"assignment_groups,omitempty"`
	Files            []any      `json:"files,omitempty"`
	Pages            []any      `json:"pages,omitempty"`
	Announcements    []any      `json:"announcements,omitempty"`
	Discussions      []any      `json:"discussions,omitempty"`
	Submissions      []any      `json:"submissions,omitempty"`
	Grades           []any      `json:"grades,omitempty"`
	ExportMeta       ExportMeta `json:"_export_meta"`
}

// authError is used to distinguish 401 errors that should abort the export.
type authError struct {
	msg string
}

func (e *authError) Error() string { return e.msg }

func isAuthError(err error) bool {
	_, ok := err.(*authError)
	return ok
}

// exportExitCode returns 0 if all sections succeeded, 8 if any failed.
func exportExitCode(result *ExportResult) int {
	if len(result.ExportMeta.SectionsFailed) > 0 {
		return 8
	}
	return 0
}

// filterSince filters a slice of items by the updated_at field.
// It keeps items whose updated_at >= since. Items without updated_at are kept.
func filterSince(items []any, since time.Time) []any {
	if since.IsZero() {
		return items
	}
	var filtered []any
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		updatedAt, exists := m["updated_at"]
		if !exists {
			filtered = append(filtered, item)
			continue
		}
		dateStr, ok := updatedAt.(string)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			filtered = append(filtered, item)
			continue
		}
		if !t.Before(since) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// fetchListRaw fetches a paginated list endpoint and returns raw []map[string]any.
// This preserves all fields from the Canvas API response.
func fetchListRaw(ctx context.Context, client *canvas.Client, path string, query url.Values, pageSize int) ([]map[string]any, int, error) {
	if query == nil {
		query = url.Values{}
	}
	if pageSize > 0 {
		query.Set("per_page", fmt.Sprintf("%d", pageSize))
	}

	var allItems []map[string]any
	reqCount := 0
	currentPath := path

	for {
		select {
		case <-ctx.Done():
			return allItems, reqCount, ctx.Err()
		default:
		}

		resp, err := client.Do(ctx, "GET", currentPath, query, nil)
		if err != nil {
			return allItems, reqCount, err
		}
		reqCount++

		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return allItems, reqCount, classifyStatusCode(resp.StatusCode)
		}

		var pageItems []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&pageItems); err != nil {
			resp.Body.Close()
			return allItems, reqCount, fmt.Errorf("decode list response: %w", err)
		}
		resp.Body.Close()

		allItems = append(allItems, pageItems...)

		// Check for next page
		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break
		}
		links := canvas.ParseLinkHeader(linkHeader)
		nextURL, ok := links["next"]
		if !ok || nextURL == "" {
			break
		}
		parsed, err := url.Parse(nextURL)
		if err != nil {
			break
		}
		currentPath = parsed.Path
		if parsed.RawQuery != "" {
			query = parsed.Query()
		} else {
			query = nil
		}
	}

	return allItems, reqCount, nil
}

// fetchSingleRaw fetches a single resource and returns it as map[string]any.
func fetchSingleRaw(ctx context.Context, client *canvas.Client, path string, query url.Values) (map[string]any, int, error) {
	resp, err := client.Do(ctx, "GET", path, query, nil)
	if err != nil {
		return nil, 1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, 1, classifyStatusCode(resp.StatusCode)
	}

	var item map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, 1, fmt.Errorf("decode response: %w", err)
	}
	return item, 1, nil
}

// classifyStatusCode returns an authError for 401, or a generic error.
func classifyStatusCode(status int) error {
	if status == 401 {
		return &authError{msg: "API error: status 401"}
	}
	return fmt.Errorf("API error: status %d", status)
}

// classifyError inspects an error and returns an authError for 401s.
func classifyError(err error) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "status 401") {
		return &authError{msg: errMsg}
	}
	return err
}

// ExportContext fetches multiple Canvas API endpoints and returns an aggregated result.
func ExportContext(ctx context.Context, client *canvas.Client, courseID string, opts ExportContextOpts) (*ExportResult, error) {
	start := time.Now()

	// Determine which sections to fetch
	sections := opts.Include
	if len(sections) == 0 {
		sections = allExportSections
	}

	// Parse --since
	var sinceTime time.Time
	if opts.Since != "" {
		var err error
		sinceTime, err = time.Parse(time.RFC3339, opts.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid --since value: %w", err)
		}
	}

	result := &ExportResult{}
	meta := ExportMeta{
		GeneratedAt:       start.UTC().Format(time.RFC3339),
		CourseID:          courseID,
		SectionsRequested: sections,
	}
	totalRequests := 0

	// Fetch each section sequentially, isolating errors
	for _, section := range sections {
		reqCount, err := fetchSection(ctx, client, courseID, section, sinceTime, result)
		totalRequests += reqCount

		if err != nil {
			if isAuthError(err) {
				return nil, err
			}
			// 403 or network error: record warning, continue
			meta.SectionsFailed = append(meta.SectionsFailed, section)
			meta.Warnings = append(meta.Warnings, fmt.Sprintf("%s: %s", section, err.Error()))
		} else {
			meta.SectionsSucceeded = append(meta.SectionsSucceeded, section)
		}
	}

	meta.RequestCount = totalRequests
	dur := time.Since(start).Milliseconds()
	if dur == 0 && totalRequests > 0 {
		dur = 1
	}
	meta.DurationMS = dur
	result.ExportMeta = meta

	return result, nil
}

// fetchSection fetches a single export section and populates the result.
// Returns the number of API requests made and any error.
func fetchSection(ctx context.Context, client *canvas.Client, courseID, section string, since time.Time, result *ExportResult) (int, error) {
	switch section {
	case "course":
		return fetchCourseSection(ctx, client, courseID, result)
	case "tabs":
		return fetchTabsSection(ctx, client, courseID, since, result)
	case "modules":
		return fetchModulesSection(ctx, client, courseID, since, result)
	case "assignments":
		return fetchAssignmentsSection(ctx, client, courseID, since, result)
	case "assignment_groups":
		return fetchAssignmentGroupsSection(ctx, client, courseID, result)
	case "files":
		return fetchFilesSection(ctx, client, courseID, since, result)
	case "pages":
		return fetchPagesSection(ctx, client, courseID, since, result)
	case "announcements":
		return fetchAnnouncementsSection(ctx, client, courseID, since, result)
	case "discussions":
		return fetchDiscussionsSection(ctx, client, courseID, since, result)
	case "submissions":
		return fetchSubmissionsSection(ctx, client, courseID, since, result)
	case "grades":
		return fetchGradesSection(ctx, client, courseID, result)
	default:
		return 0, fmt.Errorf("unknown section: %s", section)
	}
}

func fetchCourseSection(ctx context.Context, client *canvas.Client, courseID string, result *ExportResult) (int, error) {
	query := url.Values{
		"include[]": {"term", "total_scores"},
	}
	course, reqCount, err := fetchSingleRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s", courseID), query)
	if err != nil {
		return reqCount, classifyError(err)
	}
	result.Course = course
	return reqCount, nil
}

func fetchTabsSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/tabs", courseID), nil, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	anyItems := make([]any, len(items))
	for i, item := range items {
		anyItems[i] = item
	}
	filtered := filterSince(anyItems, since)
	if len(filtered) > 0 {
		result.Tabs = filtered
	}
	return reqCount, nil
}

func fetchModulesSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	query := url.Values{
		"include[]": {"items", "content_details"},
	}
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/modules", courseID), query, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	anyItems := make([]any, len(items))
	for i, item := range items {
		itemsList, hasItems := item["items"].([]any)
		itemsCountFloat, hasCount := item["items_count"].(float64)
		itemsCount := 0
		if hasCount {
			itemsCount = int(itemsCountFloat)
		}

		if (!hasItems || len(itemsList) == 0) && itemsCount > 0 {
			moduleIDFloat, ok := item["id"].(float64)
			if ok {
				moduleID := int(moduleIDFloat)
				modItems, rc, modErr := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/modules/%d/items", courseID, moduleID), nil, 100)
				reqCount += rc
				if modErr == nil {
					var anyModItems []any
					for _, mi := range modItems {
						anyModItems = append(anyModItems, mi)
					}
					item["items"] = anyModItems
				}
			}
		}

		anyItems[i] = item
	}
	filtered := filterSince(anyItems, since)
	if len(filtered) > 0 {
		result.Modules = filtered
	}
	return reqCount, nil
}

func fetchAssignmentsSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	query := url.Values{
		"order_by": {"due_at"},
	}
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/assignments", courseID), query, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	anyItems := make([]any, len(items))
	for i, item := range items {
		anyItems[i] = item
	}
	filtered := filterSince(anyItems, since)
	if len(filtered) > 0 {
		result.Assignments = filtered
	}
	return reqCount, nil
}

func fetchAssignmentGroupsSection(ctx context.Context, client *canvas.Client, courseID string, result *ExportResult) (int, error) {
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/assignment_groups", courseID), nil, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	if len(items) > 0 {
		anyItems := make([]any, len(items))
		for i, item := range items {
			anyItems[i] = item
		}
		result.AssignmentGroups = anyItems
	}
	return reqCount, nil
}

func fetchFilesSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/files", courseID), nil, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	anyItems := make([]any, len(items))
	for i, item := range items {
		anyItems[i] = item
	}
	filtered := filterSince(anyItems, since)
	if len(filtered) > 0 {
		result.Files = filtered
	}
	return reqCount, nil
}

func fetchPagesSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	// First get page list
	pages, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/pages", courseID), nil, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}

	// Fetch body for each page
	var fullPages []any
	for _, p := range pages {
		pageURL, _ := p["url"].(string)
		if pageURL == "" {
			fullPages = append(fullPages, p)
			continue
		}
		page, rc, err := fetchSingleRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/pages/%s", courseID, pageURL), nil)
		reqCount += rc
		if err != nil {
			// Non-fatal: include page without body
			fullPages = append(fullPages, p)
			continue
		}
		fullPages = append(fullPages, page)
	}

	filtered := filterSince(fullPages, since)
	if len(filtered) > 0 {
		result.Pages = filtered
	}
	return reqCount, nil
}

func fetchAnnouncementsSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	query := url.Values{
		"context_codes[]": {fmt.Sprintf("course_%s", courseID)},
	}
	items, reqCount, err := fetchListRaw(ctx, client, "/api/v1/announcements", query, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	anyItems := make([]any, len(items))
	for i, item := range items {
		anyItems[i] = item
	}
	filtered := filterSince(anyItems, since)
	if len(filtered) > 0 {
		result.Announcements = filtered
	}
	return reqCount, nil
}

func fetchDiscussionsSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	query := url.Values{
		"only_announcements": {"false"},
	}
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/discussion_topics", courseID), query, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	anyItems := make([]any, len(items))
	for i, item := range items {
		anyItems[i] = item
	}
	filtered := filterSince(anyItems, since)
	if len(filtered) > 0 {
		result.Discussions = filtered
	}
	return reqCount, nil
}

func fetchSubmissionsSection(ctx context.Context, client *canvas.Client, courseID string, since time.Time, result *ExportResult) (int, error) {
	query := url.Values{
		"student_ids[]": {"all"},
	}
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/students/submissions", courseID), query, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	anyItems := make([]any, len(items))
	for i, item := range items {
		anyItems[i] = item
	}
	filtered := filterSince(anyItems, since)
	if len(filtered) > 0 {
		result.Submissions = filtered
	}
	return reqCount, nil
}

func fetchGradesSection(ctx context.Context, client *canvas.Client, courseID string, result *ExportResult) (int, error) {
	query := url.Values{
		"user_id[]": {"self"},
		"include[]": {"total_scores"},
	}
	items, reqCount, err := fetchListRaw(ctx, client, fmt.Sprintf("/api/v1/courses/%s/enrollments", courseID), query, 100)
	if err != nil {
		return reqCount, classifyError(err)
	}
	if len(items) > 0 {
		anyItems := make([]any, len(items))
		for i, item := range items {
			anyItems[i] = item
		}
		result.Grades = anyItems
	}
	return reqCount, nil
}

// newCoursesExportContextCmd returns `courses export-context`.
func newCoursesExportContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-context",
		Short: "Export course context to JSON",
		Long: `Aggregates data from multiple Canvas API endpoints into a single JSON output
for offline review and agent consumption. Sections are fetched sequentially with
error isolation — a failure in one section does not abort others.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			outPath, _ := cmd.Flags().GetString("out")
			includeStr, _ := cmd.Flags().GetString("include")
			since, _ := cmd.Flags().GetString("since")

			var include []string
			if includeStr != "" {
				include = strings.Split(includeStr, ",")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			result, err := ExportContext(cmd.Context(), client, courseID, ExportContextOpts{
				Include: include,
				Since:   since,
			})
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_AUTH_ERROR",
						Message:  err.Error(),
						Category: "auth",
					}, "courses.export-context")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			// Determine output destination
			w := cmd.OutOrStdout()
			if outPath != "" {
				f, fileErr := os.Create(outPath)
				if fileErr != nil {
					return fmt.Errorf("create output file: %w", fileErr)
				}
				defer f.Close()
				w = f
			}

			if jsonMode {
				env := output.NewSuccess(result, "courses.export-context", canvas.Meta{
					Profile:      cfg.Profile,
					BaseURL:      cfg.BaseURL,
					DurationMS:   result.ExportMeta.DurationMS,
					RequestCount: result.ExportMeta.RequestCount,
					Warnings:     result.ExportMeta.Warnings,
				})
				return output.WriteJSON(w, env, false)
			}

			// Raw JSON mode (no envelope)
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		},
	}

	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("out", "", "output file path (default: stdout)")
	cmd.Flags().String("include", "", "comma-separated list of sections to include")
	cmd.Flags().String("since", "", "only include items updated after this date (ISO 8601)")
	cmd.Flags().String("download-files", "", "directory to download file attachments into")

	return cmd
}
