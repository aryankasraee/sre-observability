// Command shop-api is a small service that fails on purpose. It exposes the
// metrics an SLO needs (request count by status, latency histogram) and an admin
// endpoint to inject errors and latency, so alerts can be tested against real
// traffic instead of trusted.
package main

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests by route and status code.",
	}, []string{"route", "code"})

	duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by route.",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"route"})

	// Chaos settings, stored as float bits so handlers read them without locks.
	errorRate atomic.Uint64
	latencyMs atomic.Uint64
)

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func instrument(route string, next http.HandlerFunc, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next(rec, r)
		elapsed := time.Since(start)
		requests.WithLabelValues(route, strconv.Itoa(rec.code)).Inc()
		duration.WithLabelValues(route).Observe(elapsed.Seconds())
		level := slog.LevelInfo
		if rec.code >= 500 {
			level = slog.LevelWarn
		}
		log.Log(r.Context(), level, "request", "route", route, "code", rec.code, "ms", elapsed.Milliseconds())
	})
}

func work(w http.ResponseWriter, _ *http.Request) {
	if ms := latencyMs.Load(); ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	if rand.Float64() < math.Float64frombits(errorRate.Load()) {
		http.Error(w, "injected failure", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte("ok"))
}

func chaos(w http.ResponseWriter, r *http.Request) {
	if v := r.URL.Query().Get("error_rate"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 0 || f > 1 {
			http.Error(w, "error_rate must be between 0 and 1", http.StatusBadRequest)
			return
		}
		errorRate.Store(math.Float64bits(f))
	}
	if v := r.URL.Query().Get("latency_ms"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			http.Error(w, "latency_ms must be a non-negative integer", http.StatusBadRequest)
			return
		}
		latencyMs.Store(n)
	}
	_, _ = w.Write([]byte("ok"))
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	prometheus.MustRegister(requests, duration)

	mux := http.NewServeMux()
	mux.Handle("/checkout", instrument("/checkout", work, log))
	mux.Handle("/catalog", instrument("/catalog", work, log))
	mux.HandleFunc("/admin/chaos", chaos) // not instrumented: it must not move the SLI
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.Handle("/metrics", promhttp.Handler())

	log.Info("listening", "addr", ":8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
