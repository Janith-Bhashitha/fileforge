// Package metrics defines the Prometheus collectors shared by the API and
// the workers. Keeping the definitions in one place means a dashboard query
// written against the API's metric names works unchanged against a worker's.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fileforge_http_requests_total",
		Help: "HTTP requests handled, by method, route pattern and status.",
	}, []string{"method", "route", "status"})

	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fileforge_http_request_duration_seconds",
		Help:    "HTTP request duration.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	// Conversions are tracked separately from HTTP because the same
	// operation runs both synchronously (via /convert) and asynchronously
	// (via a worker), and the interesting question — "is pdf-compress
	// slow/failing?" — spans both.
	ConversionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fileforge_conversions_total",
		Help: "Conversions attempted, by operation and outcome.",
	}, []string{"operation", "outcome"})

	ConversionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fileforge_conversion_duration_seconds",
		Help:    "Conversion processing duration, by operation.",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	}, []string{"operation"})

	JobItemsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fileforge_job_items_processed_total",
		Help: "Job items processed by workers, by operation and terminal status.",
	}, []string{"operation", "status"})

	JobRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fileforge_job_retries_total",
		Help: "Job item processing attempts that failed and were retried.",
	}, []string{"operation"})

	QueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fileforge_queue_depth",
		Help: "Number of messages currently on each Redis stream.",
	}, []string{"stream"})
)
