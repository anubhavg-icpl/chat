package admin

import (
	"context"
	"fmt"
	"net/http"
)

// HTTP method constants used by the client.
const (
	httpGET    = http.MethodGet
	httpPOST   = http.MethodPost
	httpPUT    = http.MethodPut
	httpPATCH  = http.MethodPatch
	httpDELETE = http.MethodDelete
)

// ListUsers retrieves all registered user accounts.
// It calls GET /user.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var out []User
	err := c.do(ctx, httpGET, "/user", nil, &out)
	return out, err
}

// createUserRequest is the request body for POST /user.
type createUserRequest struct {
	ScreenName string `json:"screen_name"`
	Password   string `json:"password"`
}

// CreateUser creates a new AIM or ICQ user account.
// It calls POST /user.
func (c *Client) CreateUser(ctx context.Context, screenName, password string) error {
	body := createUserRequest{ScreenName: screenName, Password: password}
	return c.do(ctx, httpPOST, "/user", body, nil)
}

// deleteUserRequest is the request body for DELETE /user.
type deleteUserRequest struct {
	ScreenName string `json:"screen_name"`
}

// DeleteUser deletes the user account identified by screenName.
// It calls DELETE /user.
func (c *Client) DeleteUser(ctx context.Context, screenName string) error {
	body := deleteUserRequest{ScreenName: screenName}
	return c.do(ctx, httpDELETE, "/user", body, nil)
}

// passwordRequest is the request body for PUT /user/password.
type passwordRequest struct {
	ScreenName string `json:"screen_name"`
	Password   string `json:"password"`
}

// ResetPassword sets a new password for the user identified by screenName.
// It calls PUT /user/password.
func (c *Client) ResetPassword(ctx context.Context, screenName, password string) error {
	body := passwordRequest{ScreenName: screenName, Password: password}
	return c.do(ctx, httpPUT, "/user/password", body, nil)
}

// accountPath builds the account path for a screen name.
func accountPath(screenName string) string {
	return fmt.Sprintf("/user/%s/account", pathEscape(screenName))
}

// GetAccount retrieves the full account details for screenName.
// It calls GET /user/{screenname}/account.
func (c *Client) GetAccount(ctx context.Context, screenName string) (Account, error) {
	var out Account
	err := c.do(ctx, httpGET, accountPath(screenName), nil, &out)
	return out, err
}

// PatchAccount applies a partial update to the account identified by
// screenName. The server returns 304 when the request results in no change;
// such a response is treated as success. It calls PATCH /user/{screenname}/account.
func (c *Client) PatchAccount(ctx context.Context, screenName string, req PatchAccountRequest) error {
	return c.do(ctx, httpPATCH, accountPath(screenName), req, nil, http.StatusNotModified)
}

// SetSuspend sets the suspended status of the account identified by screenName.
// Use an empty string to clear suspension, or one of "deleted", "expired",
// "suspended", or "suspended_age".
func (c *Client) SetSuspend(ctx context.Context, screenName, status string) error {
	statusCopy := status
	return c.PatchAccount(ctx, screenName, PatchAccountRequest{SuspendedStatus: &statusCopy})
}

// ToggleBot sets the bot flag of the account identified by screenName.
func (c *Client) ToggleBot(ctx context.Context, screenName string, isBot bool) error {
	return c.PatchAccount(ctx, screenName, PatchAccountRequest{IsBot: &isBot})
}
