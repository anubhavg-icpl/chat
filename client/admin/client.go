// Package admin provides a client library for the Open OSCAR Server
// Management API (the HTTP management server that runs on port 8080 by
// default).
//
// The client is implemented using only the Go standard library. Create a
// client with [New] and invoke the typed methods to interact with users,
// sessions, chat rooms, the buddy directory, Web API keys, and more.
package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is the Management API client. It is safe for concurrent use by
// multiple goroutines.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// Option configures a [Client].
type Option func(*Client)

// WithHTTPClient sets the underlying *http.Client used for requests. If h is
// nil the default client is retained.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithTimeout sets the timeout applied to requests made by the client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// New returns a [Client] targeting the Management API at baseURL. baseURL must
// include a scheme and host, for example "http://127.0.0.1:8080".
func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL %q: %w", baseURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid base URL %q: must include scheme and host", baseURL)
	}
	c := &Client{
		baseURL:    u,
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// ErrorResponse describes a non-success (non-2xx) response from the API. It
// carries the HTTP status code and the raw response body.
type ErrorResponse struct {
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int
	// Body is the raw response body returned by the server.
	Body string
}

// Error implements the error interface.
func (e *ErrorResponse) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("server returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("server returned HTTP %d: %s", e.StatusCode, body)
}

// do performs an HTTP request against path (which must begin with "/" and may
// contain URL-escaped segments), sending body as JSON when non-nil and decoding
// a JSON response into out when non-nil. A status code outside the 2xx range
// (and not listed in extraOK) produces an *ErrorResponse.
func (c *Client) do(ctx context.Context, method, path string, body any, out any, extraOK ...int) error {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}

	rel, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("invalid request path %q: %w", path, err)
	}
	u := c.baseURL.ResolveReference(rel)

	req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if out != nil {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	for _, code := range extraOK {
		if resp.StatusCode == code {
			ok = true
			break
		}
	}
	if !ok {
		return &ErrorResponse{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response body: %w", err)
		}
	}
	return nil
}

// pathEscape escapes a single URL path segment.
func pathEscape(s string) string {
	return url.PathEscape(s)
}
