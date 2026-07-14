package main

import (
	"io"
	"net/http"
)

// NewServer returns an http.Handler that exposes the exporter's HTTP surface:
//
//   - GET /metrics  - Prometheus text exposition (200 text/plain)
//   - GET /healthz  - liveness probe (200 "ok")
//
// Any other path (or method) falls through to the default mux, which responds
// with 404 (or 405 for a known path with an unsupported method).
func NewServer(collector *Collector) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		collector.WritePrometheus(w)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok")
	})
	return mux
}
