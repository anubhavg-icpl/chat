package admin

import (
	"context"
	"fmt"
)

// ListSessions retrieves all active user sessions.
// It calls GET /session.
func (c *Client) ListSessions(ctx context.Context) (SessionList, error) {
	var out SessionList
	err := c.do(ctx, httpGET, "/session", nil, &out)
	return out, err
}

// GetSessions retrieves the active sessions for a specific screen name or UIN.
// It calls GET /session/{screenname}.
func (c *Client) GetSessions(ctx context.Context, screenName string) (SessionList, error) {
	var out SessionList
	path := fmt.Sprintf("/session/%s", pathEscape(screenName))
	err := c.do(ctx, httpGET, path, nil, &out)
	return out, err
}

// KickSession disconnects any active sessions for screenName.
// It calls DELETE /session/{screenname}.
func (c *Client) KickSession(ctx context.Context, screenName string) error {
	path := fmt.Sprintf("/session/%s", pathEscape(screenName))
	return c.do(ctx, httpDELETE, path, nil, nil)
}
