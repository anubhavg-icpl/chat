package admin

import "context"

// instantMessageRequest is the request body for POST /instant-message.
type instantMessageRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}

// SendIM sends an instant message from one screen name to another. No error is
// raised if the recipient does not exist or is offline; the sender screen name
// does not need to exist. It calls POST /instant-message.
func (c *Client) SendIM(ctx context.Context, from, to, text string) error {
	body := instantMessageRequest{From: from, To: to, Text: text}
	return c.do(ctx, httpPOST, "/instant-message", body, nil)
}
