package admin

import "context"

// GetVersion retrieves build information for the running server.
// It calls GET /version.
func (c *Client) GetVersion(ctx context.Context) (Version, error) {
	var out Version
	err := c.do(ctx, httpGET, "/version", nil, &out)
	return out, err
}
