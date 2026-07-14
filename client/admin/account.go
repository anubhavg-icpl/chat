package admin

import (
	"context"
	"fmt"
)

// ListLinkedAccounts retrieves the screen names linked to screenName.
// It calls GET /user/{screenname}/linked-account.
func (c *Client) ListLinkedAccounts(ctx context.Context, screenName string) (LinkedAccounts, error) {
	var out LinkedAccounts
	path := fmt.Sprintf("/user/%s/linked-account", pathEscape(screenName))
	err := c.do(ctx, httpGET, path, nil, &out)
	return out, err
}

// addLinkedAccountRequest is the request body for POST
// /user/{screenname}/linked-account.
type addLinkedAccountRequest struct {
	LinkedScreenName string `json:"linked_screen_name"`
}

// AddLinkedAccount links linkedScreenName to the primary account screenName.
// It calls POST /user/{screenname}/linked-account.
func (c *Client) AddLinkedAccount(ctx context.Context, screenName, linkedScreenName string) error {
	body := addLinkedAccountRequest{LinkedScreenName: linkedScreenName}
	path := fmt.Sprintf("/user/%s/linked-account", pathEscape(screenName))
	return c.do(ctx, httpPOST, path, body, nil)
}

// RemoveLinkedAccount removes the link between screenName and linkedScreenName.
// It calls DELETE /user/{screenname}/linked-account/{linked_screenname}.
func (c *Client) RemoveLinkedAccount(ctx context.Context, screenName, linkedScreenName string) error {
	path := fmt.Sprintf("/user/%s/linked-account/%s", pathEscape(screenName), pathEscape(linkedScreenName))
	return c.do(ctx, httpDELETE, path, nil, nil)
}
