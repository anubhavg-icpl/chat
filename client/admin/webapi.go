package admin

import (
	"context"
	"fmt"
)

// ListWebAPIKeys retrieves all Web API keys.
// It calls GET /admin/webapi/keys.
func (c *Client) ListWebAPIKeys(ctx context.Context) ([]WebAPIKey, error) {
	var out []WebAPIKey
	err := c.do(ctx, httpGET, "/admin/webapi/keys", nil, &out)
	return out, err
}

// GetWebAPIKey retrieves a single Web API key by its developer ID.
// It calls GET /admin/webapi/keys/{id}.
func (c *Client) GetWebAPIKey(ctx context.Context, devID string) (WebAPIKey, error) {
	var out WebAPIKey
	path := fmt.Sprintf("/admin/webapi/keys/%s", pathEscape(devID))
	err := c.do(ctx, httpGET, path, nil, &out)
	return out, err
}

// CreateWebAPIKey creates a new Web API key. The returned [WebAPIKey].DevKey
// contains the actual API key value, which is only shown at creation time.
// It calls POST /admin/webapi/keys.
func (c *Client) CreateWebAPIKey(ctx context.Context, req CreateWebAPIKeyRequest) (WebAPIKey, error) {
	var out WebAPIKey
	err := c.do(ctx, httpPOST, "/admin/webapi/keys", req, &out)
	return out, err
}

// UpdateWebAPIKey updates an existing Web API key by developer ID. Only fields
// with non-nil values in req are applied. It calls PUT /admin/webapi/keys/{id}.
func (c *Client) UpdateWebAPIKey(ctx context.Context, devID string, req UpdateWebAPIKeyRequest) (WebAPIKey, error) {
	var out WebAPIKey
	path := fmt.Sprintf("/admin/webapi/keys/%s", pathEscape(devID))
	err := c.do(ctx, httpPUT, path, req, &out)
	return out, err
}

// DeleteWebAPIKey permanently deletes the Web API key identified by devID.
// It calls DELETE /admin/webapi/keys/{id}.
func (c *Client) DeleteWebAPIKey(ctx context.Context, devID string) error {
	path := fmt.Sprintf("/admin/webapi/keys/%s", pathEscape(devID))
	return c.do(ctx, httpDELETE, path, nil, nil)
}

// ToggleWebAPIKey enables or disables the Web API key identified by devID.
func (c *Client) ToggleWebAPIKey(ctx context.Context, devID string, active bool) (WebAPIKey, error) {
	return c.UpdateWebAPIKey(ctx, devID, UpdateWebAPIKeyRequest{IsActive: &active})
}
