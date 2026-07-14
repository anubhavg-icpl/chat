// Command metrics-exporter periodically scrapes the Open OSCAR Server
// Management API and exposes the results as Prometheus-format metrics on an
// HTTP /metrics endpoint, allowing operators to monitor the server with
// Prometheus/Grafana.
//
// Configuration is via environment variables:
//
//	OSCAR_API                - Management API base URL (default http://127.0.0.1:8080)
//	METRICS_ADDR             - address to serve /metrics on (default :9090)
//	METRICS_SCRAPE_INTERVAL  - poll interval (default 15s)
//
// The exporter uses only the Go standard library plus the internal
// github.com/mk6i/open-oscar-server/client/admin client. It hand-writes the
// Prometheus text exposition format rather than depending on a metrics library.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mk6i/open-oscar-server/client/admin"
)

// buildVersion is the exporter version. It may be overridden at build time via
// -ldflags "-X main.buildVersion=...".
var buildVersion = "dev"

const (
	defaultAPIBaseURL    = "http://127.0.0.1:8080"
	defaultMetricsAddr   = ":9090"
	defaultScrapePeriod  = 15 * time.Second
	scrapeRequestTimeout = 30 * time.Second
	shutdownTimeout      = 10 * time.Second
)

func main() {
	apiBaseURL := getenv("OSCAR_API", defaultAPIBaseURL)
	metricsAddr := getenv("METRICS_ADDR", defaultMetricsAddr)

	interval := defaultScrapePeriod
	if v := os.Getenv("METRICS_SCRAPE_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("invalid METRICS_SCRAPE_INTERVAL %q: %v", v, err)
		}
		if d <= 0 {
			log.Fatalf("METRICS_SCRAPE_INTERVAL must be positive, got %v", d)
		}
		interval = d
	}

	client, err := admin.New(apiBaseURL, admin.WithTimeout(scrapeRequestTimeout))
	if err != nil {
		log.Fatalf("invalid OSCAR_API %q: %v", apiBaseURL, err)
	}

	collector := NewCollector(client)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Perform an initial scrape so /metrics is populated immediately.
	if err := collector.Scrape(ctx); err != nil {
		log.Printf("initial scrape failed: %v", err)
	}

	go runCollector(ctx, collector, interval)

	server := &http.Server{
		Addr:    metricsAddr,
		Handler: NewServer(collector),
	}

	go func() {
		log.Printf("metrics-exporter listening on %s (scraping %s every %s)", metricsAddr, apiBaseURL, interval)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
}

// runCollector polls the API every interval until ctx is cancelled, logging any
// scrape errors without terminating.
func runCollector(ctx context.Context, collector *Collector, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := collector.Scrape(ctx); err != nil {
				log.Printf("scrape failed: %v", err)
			}
		}
	}
}

// getenv returns the value of key if set (and non-empty), otherwise def.
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
