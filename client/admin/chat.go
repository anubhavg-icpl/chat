package admin

import "context"

// ListPublicRooms retrieves all public AIM chat rooms.
// It calls GET /chat/room/public.
func (c *Client) ListPublicRooms(ctx context.Context) ([]ChatRoom, error) {
	var out []ChatRoom
	err := c.do(ctx, httpGET, "/chat/room/public", nil, &out)
	return out, err
}

// ListPrivateRooms retrieves all private AIM chat rooms.
// It calls GET /chat/room/private.
func (c *Client) ListPrivateRooms(ctx context.Context) ([]ChatRoom, error) {
	var out []ChatRoom
	err := c.do(ctx, httpGET, "/chat/room/private", nil, &out)
	return out, err
}

// createRoomRequest is the request body for POST /chat/room/public.
type createRoomRequest struct {
	Name string `json:"name"`
}

// CreatePublicRoom creates a new public chat room named name.
// It calls POST /chat/room/public.
func (c *Client) CreatePublicRoom(ctx context.Context, name string) error {
	body := createRoomRequest{Name: name}
	return c.do(ctx, httpPOST, "/chat/room/public", body, nil)
}

// deleteRoomsRequest is the request body for DELETE /chat/room/public.
type deleteRoomsRequest struct {
	Names []string `json:"names"`
}

// DeletePublicRooms deletes one or more public chat rooms by name.
// It calls DELETE /chat/room/public.
func (c *Client) DeletePublicRooms(ctx context.Context, names []string) error {
	body := deleteRoomsRequest{Names: names}
	return c.do(ctx, httpDELETE, "/chat/room/public", body, nil)
}
