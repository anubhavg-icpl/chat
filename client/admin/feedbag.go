package admin

import (
	"context"
	"fmt"
)

// GetFeedbag retrieves the buddy list for screenName, grouped by group ID.
// It calls GET /feedbag/{screen_name}/group.
func (c *Client) GetFeedbag(ctx context.Context, screenName string) ([]FeedbagGroup, error) {
	var out []FeedbagGroup
	path := fmt.Sprintf("/feedbag/%s/group", pathEscape(screenName))
	err := c.do(ctx, httpGET, path, nil, &out)
	return out, err
}

// CreateFeedbagGroup creates a buddy list group named groupName for screenName.
// The operation is idempotent. It calls PUT /feedbag/{screen_name}/group/{group_name}.
func (c *Client) CreateFeedbagGroup(ctx context.Context, screenName, groupName string) (FeedbagGroupResult, error) {
	var out FeedbagGroupResult
	path := fmt.Sprintf("/feedbag/%s/group/%s", pathEscape(screenName), pathEscape(groupName))
	err := c.do(ctx, httpPUT, path, nil, &out)
	return out, err
}

// AddBuddy adds buddyScreenName to the group identified by groupID for
// screenName. The operation is idempotent. It calls
// PUT /feedbag/{screen_name}/group/{group_id}/buddy/{buddy_screen_name}.
func (c *Client) AddBuddy(ctx context.Context, screenName string, groupID int, buddyScreenName string) (FeedbagBuddyResult, error) {
	var out FeedbagBuddyResult
	path := fmt.Sprintf("/feedbag/%s/group/%d/buddy/%s", pathEscape(screenName), groupID, pathEscape(buddyScreenName))
	err := c.do(ctx, httpPUT, path, nil, &out)
	return out, err
}

// RemoveBuddy removes buddyScreenName from the group identified by groupID for
// screenName. It calls
// DELETE /feedbag/{screen_name}/group/{group_id}/buddy/{buddy_screen_name}.
func (c *Client) RemoveBuddy(ctx context.Context, screenName string, groupID int, buddyScreenName string) error {
	path := fmt.Sprintf("/feedbag/%s/group/%d/buddy/%s", pathEscape(screenName), groupID, pathEscape(buddyScreenName))
	return c.do(ctx, httpDELETE, path, nil, nil)
}
