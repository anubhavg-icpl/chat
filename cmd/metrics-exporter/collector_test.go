package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mk6i/open-oscar-server/client/admin"
)

// jsonHandler returns an http.HandlerFunc that encodes v as JSON.
func jsonHandler(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
}

// newAPIServer starts an httptest.Management-API stub that dispatches on
// "METHOD /path" to the supplied handlers. It is closed automatically.
func newAPIServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestWritePrometheus_Format(t *testing.T) {
	c := &Collector{}
	c.snap = snapshot{
		up:             1,
		usersTotal:     5,
		sessionsTotal:  3,
		publicRooms:    2,
		privateRooms:   1,
		webAPIKeys:     4,
		categories:     6,
		scrapeDuration: 0.125,
		serverVersion:  "1.2.3",
		serverCommit:   "abc123",
		serverDate:     "2026-01-01T00:00:00Z",
	}

	var buf bytes.Buffer
	c.WritePrometheus(&buf)
	out := buf.String()

	// HELP/TYPE metadata present for a representative metric.
	assert.Contains(t, out, "# HELP oscar_api_up 1 if the last scrape of the OSCAR Management API succeeded, 0 otherwise.")
	assert.Contains(t, out, "# TYPE oscar_api_up gauge")

	// Each gauge renders with its expected value.
	assert.Contains(t, out, "oscar_api_up 1")
	assert.Contains(t, out, "oscar_users_total 5")
	assert.Contains(t, out, "oscar_sessions_total 3")
	assert.Contains(t, out, "oscar_chat_rooms_public_total 2")
	assert.Contains(t, out, "oscar_chat_rooms_private_total 1")
	assert.Contains(t, out, "oscar_webapi_keys_total 4")
	assert.Contains(t, out, "oscar_directory_categories_total 6")
	assert.Contains(t, out, "oscar_scrape_duration_seconds 0.125")

	// Labelled gauges: oscar_info carries server build labels (sorted), set to 1.
	assert.Contains(t, out, `oscar_info{commit="abc123",date="2026-01-01T00:00:00Z",version="1.2.3"} 1`)
	// Build-info gauge uses the package buildVersion variable.
	assert.Contains(t, out, fmt.Sprintf(`oscar_exporter_build_info{version=%q} 1`, buildVersion))

	// Parse-check a couple of lines for exact (not just substring) formatting.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Contains(t, lines, "oscar_users_total 5")
	assert.Contains(t, lines, "oscar_api_up 1")
	assert.Contains(t, lines, `oscar_info{commit="abc123",date="2026-01-01T00:00:00Z",version="1.2.3"} 1`)
}

func TestWritePrometheus_FloatRendering(t *testing.T) {
	tests := []struct {
		name string
		v    float64
		want string
	}{
		{"integer", 5.0, "5"},
		{"zero", 0.0, "0"},
		{"fraction", 0.125, "0.125"},
		{"one and a half", 1.5, "1.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatFloat(tt.v))
		})
	}
}

func TestEscapeLabelValue(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "plain", "plain"},
		{"empty", "", ""},
		{"double quote", `a"b`, `a\"b`},
		{"backslash", `a\b`, `a\\b`},
		{"newline", "a\nb", `a\nb`},
		{"combined", "a\"b\\c\nd", `a\"b\\c\nd`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeLabelValue(tt.in))
		})
	}
}

func TestWritePrometheus_LabelValuesEscaped(t *testing.T) {
	c := &Collector{}
	c.snap = snapshot{
		up:            1,
		serverVersion: `v"1`,
		serverCommit:  `c\2`,
		serverDate:    "d\n3",
	}
	var buf bytes.Buffer
	c.WritePrometheus(&buf)
	assert.Contains(t, buf.String(), `oscar_info{commit="c\\2",date="d\n3",version="v\"1"} 1`)
}

func TestScrape_Success(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"GET /version":            jsonHandler(admin.Version{Version: "1.2.3", Commit: "abc", Date: "2026-01-01T00:00:00Z"}),
		"GET /user":               jsonHandler([]admin.User{{ID: "u1"}, {ID: "u2"}}),
		"GET /session":            jsonHandler(admin.SessionList{Count: 3, Sessions: []admin.Session{{}, {}, {}}}),
		"GET /chat/room/public":   jsonHandler([]admin.ChatRoom{{}, {}}),
		"GET /chat/room/private":  jsonHandler([]admin.ChatRoom{{}}),
		"GET /admin/webapi/keys":  jsonHandler([]admin.WebAPIKey{{}, {}, {}, {}}),
		"GET /directory/category": jsonHandler([]admin.DirectoryCategory{{}, {}, {}, {}, {}, {}}),
	}
	srv := newAPIServer(t, routes)

	client, err := admin.New(srv.URL)
	require.NoError(t, err)

	c := NewCollector(client)
	require.NoError(t, c.Scrape(context.Background()))

	snap := c.Snapshot()
	assert.Equal(t, 1.0, snap.up)
	assert.Equal(t, "1.2.3", snap.serverVersion)
	assert.Equal(t, "abc", snap.serverCommit)
	assert.Equal(t, "2026-01-01T00:00:00Z", snap.serverDate)
	assert.Equal(t, 2.0, snap.usersTotal)
	assert.Equal(t, 3.0, snap.sessionsTotal)
	assert.Equal(t, 2.0, snap.publicRooms)
	assert.Equal(t, 1.0, snap.privateRooms)
	assert.Equal(t, 4.0, snap.webAPIKeys)
	assert.Equal(t, 6.0, snap.categories)
	assert.Greater(t, snap.scrapeDuration, 0.0)
}

func TestScrape_Failure_SetsDownAndKeepsLastGood(t *testing.T) {
	var fail atomic.Bool

	// /version is the failure trigger; everything else always succeeds.
	routes := map[string]http.HandlerFunc{
		"GET /version": func(w http.ResponseWriter, r *http.Request) {
			if fail.Load() {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			jsonHandler(admin.Version{Version: "1.2.3", Commit: "abc", Date: "2026-01-01T00:00:00Z"})(w, r)
		},
		"GET /user":               jsonHandler([]admin.User{{ID: "u1"}, {ID: "u2"}}),
		"GET /session":            jsonHandler(admin.SessionList{Count: 3, Sessions: []admin.Session{{}, {}, {}}}),
		"GET /chat/room/public":   jsonHandler([]admin.ChatRoom{{}, {}}),
		"GET /chat/room/private":  jsonHandler([]admin.ChatRoom{{}}),
		"GET /admin/webapi/keys":  jsonHandler([]admin.WebAPIKey{{}, {}, {}, {}}),
		"GET /directory/category": jsonHandler([]admin.DirectoryCategory{{}, {}, {}, {}, {}, {}}),
	}
	srv := newAPIServer(t, routes)

	client, err := admin.New(srv.URL)
	require.NoError(t, err)
	c := NewCollector(client)

	// First scrape succeeds.
	require.NoError(t, c.Scrape(context.Background()))
	good := c.Snapshot()
	require.Equal(t, 1.0, good.up)
	require.Equal(t, 2.0, good.usersTotal)

	// Second scrape fails: up drops to 0 but last good values are retained.
	fail.Store(true)
	err = c.Scrape(context.Background())
	require.Error(t, err)

	snap := c.Snapshot()
	assert.Equal(t, 0.0, snap.up)
	assert.Equal(t, 2.0, snap.usersTotal, "last good users retained")
	assert.Equal(t, 3.0, snap.sessionsTotal, "last good sessions retained")
	assert.Equal(t, "1.2.3", snap.serverVersion, "last good version retained")
	assert.Greater(t, snap.scrapeDuration, 0.0, "scrape duration still recorded")

	// The rendered output must reflect the down state but keep last good counts.
	var buf bytes.Buffer
	c.WritePrometheus(&buf)
	body := buf.String()
	assert.Contains(t, body, "oscar_api_up 0")
	assert.Contains(t, body, "oscar_users_total 2")
}
