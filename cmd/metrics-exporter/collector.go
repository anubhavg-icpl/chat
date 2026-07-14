package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mk6i/open-oscar-server/client/admin"
)

// Collector caches a snapshot of metrics scraped from the Management API and
// renders them in Prometheus text exposition format. It is safe for concurrent
// use: the collector goroutine updates the snapshot while HTTP handlers read it.
type Collector struct {
	client *admin.Client
	mu     sync.Mutex
	snap   snapshot
}

// snapshot holds the most recent values for every exposed metric.
type snapshot struct {
	up             float64
	usersTotal     float64
	sessionsTotal  float64
	publicRooms    float64
	privateRooms   float64
	webAPIKeys     float64
	categories     float64
	scrapeDuration float64
	serverVersion  string
	serverCommit   string
	serverDate     string
}

// NewCollector returns a Collector that scrapes the API via client.
func NewCollector(client *admin.Client) *Collector {
	return &Collector{client: client}
}

// Snapshot returns a copy of the cached metrics snapshot.
func (c *Collector) Snapshot() snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snap
}

// Scrape polls every Management API endpoint the exporter depends on, times the
// attempt, and updates the cached snapshot. On any error it sets oscar_api_up
// to 0 and retains the last good values for the resource gauges while still
// updating the scrape duration. It returns the first error encountered.
func (c *Collector) Scrape(ctx context.Context) error {
	start := time.Now()

	var firstErr error
	note := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	ver, err := c.client.GetVersion(ctx)
	note(err)
	users, err := c.client.ListUsers(ctx)
	note(err)
	sessions, err := c.client.ListSessions(ctx)
	note(err)
	publicRooms, err := c.client.ListPublicRooms(ctx)
	note(err)
	privateRooms, err := c.client.ListPrivateRooms(ctx)
	note(err)
	keys, err := c.client.ListWebAPIKeys(ctx)
	note(err)
	categories, err := c.client.ListCategories(ctx)
	note(err)

	duration := time.Since(start).Seconds()

	c.mu.Lock()
	defer c.mu.Unlock()
	c.snap.scrapeDuration = duration
	if firstErr != nil {
		c.snap.up = 0
		return firstErr
	}

	c.snap.up = 1
	c.snap.serverVersion = ver.Version
	c.snap.serverCommit = ver.Commit
	c.snap.serverDate = ver.Date
	c.snap.usersTotal = float64(len(users))
	c.snap.sessionsTotal = float64(sessions.Count)
	c.snap.publicRooms = float64(len(publicRooms))
	c.snap.privateRooms = float64(len(privateRooms))
	c.snap.webAPIKeys = float64(len(keys))
	c.snap.categories = float64(len(categories))
	return nil
}

// labelPair is an ordered name/value label for a Prometheus metric line.
type labelPair struct {
	name  string
	value string
}

// WritePrometheus emits the cached metrics in Prometheus text exposition format
// to w. HELP and TYPE metadata lines are emitted for each metric, followed by
// the metric sample line(s).
func (c *Collector) WritePrometheus(w io.Writer) {
	c.mu.Lock()
	snap := c.snap
	c.mu.Unlock()

	header := func(name, help string) {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", name, help, name)
	}

	header("oscar_api_up", "1 if the last scrape of the OSCAR Management API succeeded, 0 otherwise.")
	writeGauge(w, "oscar_api_up", nil, snap.up)

	header("oscar_users_total", "Total number of registered user accounts.")
	writeGauge(w, "oscar_users_total", nil, snap.usersTotal)

	header("oscar_sessions_total", "Number of active user sessions.")
	writeGauge(w, "oscar_sessions_total", nil, snap.sessionsTotal)

	header("oscar_chat_rooms_public_total", "Total number of public chat rooms.")
	writeGauge(w, "oscar_chat_rooms_public_total", nil, snap.publicRooms)

	header("oscar_chat_rooms_private_total", "Total number of private chat rooms.")
	writeGauge(w, "oscar_chat_rooms_private_total", nil, snap.privateRooms)

	header("oscar_webapi_keys_total", "Total number of Web API keys.")
	writeGauge(w, "oscar_webapi_keys_total", nil, snap.webAPIKeys)

	header("oscar_directory_categories_total", "Total number of directory keyword categories.")
	writeGauge(w, "oscar_directory_categories_total", nil, snap.categories)

	header("oscar_scrape_duration_seconds", "Duration of the last Management API scrape in seconds.")
	writeGauge(w, "oscar_scrape_duration_seconds", nil, snap.scrapeDuration)

	header("oscar_info", "Build information for the running OSCAR server.")
	writeGauge(w, "oscar_info", []labelPair{
		{"commit", snap.serverCommit},
		{"date", snap.serverDate},
		{"version", snap.serverVersion},
	}, 1)

	header("oscar_exporter_build_info", "Build information for the OSCAR metrics exporter.")
	writeGauge(w, "oscar_exporter_build_info", []labelPair{{"version", buildVersion}}, 1)
}

// writeGauge writes a single gauge sample line. When labels is empty the line is
// emitted without a label set. Label values are escaped per the Prometheus text
// format; labels are expected to be provided in the desired (typically sorted)
// order.
func writeGauge(w io.Writer, name string, labels []labelPair, value float64) {
	if len(labels) == 0 {
		fmt.Fprintf(w, "%s %s\n", name, formatFloat(value))
		return
	}
	var b strings.Builder
	for i, lp := range labels {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(lp.name)
		b.WriteString(`="`)
		b.WriteString(escapeLabelValue(lp.value))
		b.WriteByte('"')
	}
	fmt.Fprintf(w, "%s{%s} %s\n", name, b.String(), formatFloat(value))
}

// formatFloat renders v using the shortest representation, matching the
// Prometheus text format's expectations for sample values.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// escapeLabelValue escapes a label value per the Prometheus text exposition
// format: backslash, double-quote, and line-feed characters are escaped.
func escapeLabelValue(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
	).Replace(s)
}
