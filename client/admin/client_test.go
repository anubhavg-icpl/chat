package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requestRecord captures the method, path, and decoded JSON body of a request.
type requestRecord struct {
	method string
	path   string
	body   map[string]any
}

// newTestServer returns an httptest.Server whose handler records each request
// and dispatches on path+method to a handler in routes, returning status and
// body.
func newTestServer(t *testing.T, routes map[string]http.HandlerFunc, rec *requestRecord) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.Path

		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &rec.body)
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
		}

		key := r.Method + " " + r.URL.Path
		h, ok := routes[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL)
	require.NoError(t, err)
	return c, srv
}

func TestNew_InvalidBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"missing scheme", "127.0.0.1:8080"},
		{"garbage", "://nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.baseURL)
			assert.Error(t, err)
		})
	}
}

func TestNew_OptionsApplied(t *testing.T) {
	custom := &http.Client{}
	c, err := New("http://127.0.0.1:8080", WithHTTPClient(custom), WithTimeout(42*time.Second))
	require.NoError(t, err)
	assert.Same(t, custom, c.httpClient)
	assert.Equal(t, 42*time.Second, c.httpClient.Timeout)
}

func TestListUsers(t *testing.T) {
	payload := []User{
		{ID: "usera", ScreenName: "Alpha", IsICQ: false, SuspendedStatus: "", IsBot: false},
		{ID: "userb", ScreenName: "Beta", IsICQ: true, SuspendedStatus: "suspended", IsBot: true},
	}
	routes := map[string]http.HandlerFunc{
		"GET /user": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	users, err := c.ListUsers(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "GET", rec.method)
	assert.Equal(t, "/user", rec.path)
	assert.Equal(t, payload, users)
}

func TestCreateUser(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"POST /user": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("User account created successfully.\n"))
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.CreateUser(context.Background(), "NewUser", "s3cret")
	require.NoError(t, err)
	assert.Equal(t, "POST", rec.method)
	assert.Equal(t, "/user", rec.path)
	assert.Equal(t, map[string]any{
		"screen_name": "NewUser",
		"password":    "s3cret",
	}, rec.body)
}

func TestDeleteUser(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"DELETE /user": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.DeleteUser(context.Background(), "GoneUser")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", rec.method)
	assert.Equal(t, "/user", rec.path)
	assert.Equal(t, map[string]any{"screen_name": "GoneUser"}, rec.body)
}

func TestKickSession(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"DELETE /session/someuser": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.KickSession(context.Background(), "someuser")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", rec.method)
	assert.Equal(t, "/session/someuser", rec.path)
	assert.Empty(t, rec.body)
}

func TestSendIM(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"POST /instant-message": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Message sent successfully.\n"))
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.SendIM(context.Background(), "sender", "recipient", "hello world")
	require.NoError(t, err)
	assert.Equal(t, "POST", rec.method)
	assert.Equal(t, "/instant-message", rec.path)
	assert.Equal(t, map[string]any{
		"from": "sender",
		"to":   "recipient",
		"text": "hello world",
	}, rec.body)
}

func TestGetVersion(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"GET /version": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Version{Version: "1.2.3", Commit: "abc123", Date: "2026-01-01T00:00:00Z"})
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	ver, err := c.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", ver.Version)
	assert.Equal(t, "abc123", ver.Commit)
	assert.Equal(t, "GET", rec.method)
	assert.Equal(t, "/version", rec.path)
}

func TestListSessions(t *testing.T) {
	want := SessionList{Count: 1, Sessions: []Session{
		{ID: "u1", ScreenName: "Alice", InstanceCount: 1, Instances: []SessionInstance{{Num: 1}}},
	}}
	routes := map[string]http.HandlerFunc{
		"GET /session": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(want)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	got, err := c.ListSessions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCreatePublicRoom(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"POST /chat/room/public": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.CreatePublicRoom(context.Background(), "Off Topic")
	require.NoError(t, err)
	assert.Equal(t, "POST", rec.method)
	assert.Equal(t, "/chat/room/public", rec.path)
	assert.Equal(t, map[string]any{"name": "Off Topic"}, rec.body)
}

func TestDeletePublicRooms(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"DELETE /chat/room/public": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.DeletePublicRooms(context.Background(), []string{"Room A", "Room B"})
	require.NoError(t, err)
	assert.Equal(t, "DELETE", rec.method)
	assert.Equal(t, map[string]any{"names": []any{"Room A", "Room B"}}, rec.body)
}

func TestPatchAccount_OnlySetFields(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"PATCH /user/alice/account": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, map[string]any{"is_bot": true}, body)
			w.WriteHeader(http.StatusNoContent)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.ToggleBot(context.Background(), "alice", true)
	require.NoError(t, err)
	assert.Equal(t, "PATCH", rec.method)
	assert.Equal(t, "/user/alice/account", rec.path)
	assert.Equal(t, map[string]any{"is_bot": true}, rec.body)
}

func TestPatchAccount_NotModified(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"PATCH /user/alice/account": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotModified)
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	err := c.SetSuspend(context.Background(), "alice", "")
	require.NoError(t, err)
}

func TestCreateWebAPIKey(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"POST /admin/webapi/keys": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "My App", body["app_name"])
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(WebAPIKey{DevID: "dev_1", DevKey: "abc", AppName: "My App", IsActive: true, RateLimit: 60})
		},
	}
	var rec requestRecord
	c, _ := newTestServer(t, routes, &rec)

	key, err := c.CreateWebAPIKey(context.Background(), CreateWebAPIKeyRequest{AppName: "My App"})
	require.NoError(t, err)
	assert.Equal(t, "dev_1", key.DevID)
	assert.Equal(t, "abc", key.DevKey)
	assert.True(t, key.IsActive)
	assert.Equal(t, "/admin/webapi/keys", rec.path)
}

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		handler func(*Client) error
	}{
		{
			name:   "ListUsers 500",
			status: http.StatusInternalServerError,
			body:   "internal server error",
			handler: func(c *Client) error {
				_, err := c.ListUsers(context.Background())
				return err
			},
		},
		{
			name:   "CreateUser 409",
			status: http.StatusConflict,
			body:   "user already exists",
			handler: func(c *Client) error {
				return c.CreateUser(context.Background(), "dup", "pw")
			},
		},
		{
			name:   "KickSession 404",
			status: http.StatusNotFound,
			body:   "session not found",
			handler: func(c *Client) error {
				return c.KickSession(context.Background(), "nobody")
			},
		},
		{
			name:   "SendIM 400",
			status: http.StatusBadRequest,
			body:   "malformed input",
			handler: func(c *Client) error {
				return c.SendIM(context.Background(), "", "", "")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := map[string]http.HandlerFunc{
				"GET /user": func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, tt.body, tt.status)
				},
				"POST /user": func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, tt.body, tt.status)
				},
				"DELETE /session/nobody": func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, tt.body, tt.status)
				},
				"POST /instant-message": func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, tt.body, tt.status)
				},
			}
			var rec requestRecord
			c, _ := newTestServer(t, routes, &rec)

			err := tt.handler(c)
			require.Error(t, err)
			var apiErr *ErrorResponse
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.status, apiErr.StatusCode)
			assert.True(t, strings.Contains(apiErr.Body, strings.TrimSpace(tt.body)))
		})
	}
}
