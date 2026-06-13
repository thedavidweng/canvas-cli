package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// GetModule returns a single module by ID.
// It sends GET /api/v1/courses/{courseID}/modules/{moduleID}.
func GetModule(ctx context.Context, client *Client, courseID, moduleID string) (Module, error) {
	return Get[Module](ctx, client, fmt.Sprintf("/api/v1/courses/%s/modules/%s", courseID, moduleID))
}

// ListModules returns all modules for a course.
// It sends GET /api/v1/courses/{courseID}/modules.
func ListModules(ctx context.Context, client *Client, courseID string, query url.Values) ([]Module, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	modules, meta, err := List[Module](ctx, client, fmt.Sprintf("/api/v1/courses/%s/modules", courseID), query, 100)
	if err != nil {
		return nil, meta, fmt.Errorf("list modules for course %s: %w", courseID, err)
	}

	return modules, meta, nil
}

// GetModuleItem returns a single module item by ID.
func GetModuleItem(ctx context.Context, client *Client, courseID, moduleID, itemID string) (ModuleItem, error) {
	return Get[ModuleItem](ctx, client, fmt.Sprintf("/api/v1/courses/%s/modules/%s/items/%s", courseID, moduleID, itemID))
}

// ListModuleItems returns all items within a module.
// It sends GET /api/v1/courses/{courseID}/modules/{moduleID}/items.
func ListModuleItems(ctx context.Context, client *Client, courseID, moduleID string, query url.Values) ([]ModuleItem, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	items, meta, err := List[ModuleItem](ctx, client, fmt.Sprintf("/api/v1/courses/%s/modules/%s/items", courseID, moduleID), query, 100)
	if err != nil {
		return nil, meta, fmt.Errorf("list module items for course %s module %s: %w", courseID, moduleID, err)
	}

	return items, meta, nil
}
