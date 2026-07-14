package admin

import (
	"context"
	"fmt"
)

// ListCategories retrieves all keyword categories.
// It calls GET /directory/category.
func (c *Client) ListCategories(ctx context.Context) ([]DirectoryCategory, error) {
	var out []DirectoryCategory
	err := c.do(ctx, httpGET, "/directory/category", nil, &out)
	return out, err
}

// createCategoryRequest is the request body for POST /directory/category.
type createCategoryRequest struct {
	Name string `json:"name"`
}

// CreateCategory creates a new keyword category named name.
// It calls POST /directory/category.
func (c *Client) CreateCategory(ctx context.Context, name string) (DirectoryCategory, error) {
	var out DirectoryCategory
	body := createCategoryRequest{Name: name}
	err := c.do(ctx, httpPOST, "/directory/category", body, &out)
	return out, err
}

// DeleteCategory deletes the keyword category identified by categoryID.
// It calls DELETE /directory/category/{id}.
func (c *Client) DeleteCategory(ctx context.Context, categoryID int) error {
	path := fmt.Sprintf("/directory/category/%d", categoryID)
	return c.do(ctx, httpDELETE, path, nil, nil)
}

// ListKeywords retrieves all keywords in the category identified by categoryID.
// It calls GET /directory/category/{id}/keyword.
func (c *Client) ListKeywords(ctx context.Context, categoryID int) ([]DirectoryKeyword, error) {
	var out []DirectoryKeyword
	path := fmt.Sprintf("/directory/category/%d/keyword", categoryID)
	err := c.do(ctx, httpGET, path, nil, &out)
	return out, err
}

// createKeywordRequest is the request body for POST /directory/keyword.
type createKeywordRequest struct {
	CategoryID uint8  `json:"category_id"`
	Name       string `json:"name"`
}

// CreateKeyword creates a new keyword named name within the category identified
// by categoryID. It calls POST /directory/keyword.
func (c *Client) CreateKeyword(ctx context.Context, categoryID int, name string) (DirectoryKeyword, error) {
	var out DirectoryKeyword
	body := createKeywordRequest{CategoryID: uint8(categoryID), Name: name}
	err := c.do(ctx, httpPOST, "/directory/keyword", body, &out)
	return out, err
}

// DeleteKeyword deletes the keyword identified by keywordID.
// It calls DELETE /directory/keyword/{id}.
func (c *Client) DeleteKeyword(ctx context.Context, keywordID int) error {
	path := fmt.Sprintf("/directory/keyword/%d", keywordID)
	return c.do(ctx, httpDELETE, path, nil, nil)
}
