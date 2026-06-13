package canvas

import (
	"context"
	"fmt"
	"net/url"
)

// GetModule returns a single module by ID.
// It sends GET /api/v1/courses/{courseID}/modules/{moduleID}.
func GetModule(ctx context.Context, client *Client, courseID, moduleID string) (Module, error) {
	var mod Module
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/modules/%s", courseID, moduleID),
		DecodeInto: &mod,
	})
	if err != nil {
		return mod, fmt.Errorf("get module %s in course %s: %w", moduleID, courseID, err)
	}

	return mod, nil
}

// ListModules returns all modules for a course.
// It sends GET /api/v1/courses/{courseID}/modules.
func ListModules(ctx context.Context, client *Client, courseID string, query url.Values) ([]Module, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	var modules []Module
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/modules", courseID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &modules,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list modules for course %s: %w", courseID, err)
	}

	return modules, meta.Pagination, nil
}

// GetModuleItem returns a single module item by ID.
func GetModuleItem(ctx context.Context, client *Client, courseID, moduleID, itemID string) (ModuleItem, error) {
	var item ModuleItem
	_, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/modules/%s/items/%s", courseID, moduleID, itemID),
		DecodeInto: &item,
	})
	if err != nil {
		return item, fmt.Errorf("get module item %s in module %s course %s: %w", itemID, moduleID, courseID, err)
	}
	return item, nil
}

// ListModuleItems returns all items within a module.
// It sends GET /api/v1/courses/{courseID}/modules/{moduleID}/items.
func ListModuleItems(ctx context.Context, client *Client, courseID, moduleID string, query url.Values) ([]ModuleItem, PaginationMeta, error) {
	if query == nil {
		query = url.Values{}
	}

	var items []ModuleItem
	meta, err := Request(ctx, client, RequestOptions{
		Method:     "GET",
		PathOrURL:  fmt.Sprintf("/api/v1/courses/%s/modules/%s/items", courseID, moduleID),
		Query:      query,
		Paginate:   true,
		PageSize:   100,
		DecodeInto: &items,
	})
	if err != nil {
		return nil, meta.Pagination, fmt.Errorf("list module items for course %s module %s: %w", courseID, moduleID, err)
	}

	return items, meta.Pagination, nil
}
