// Package metrics provides Prometheus metrics collection
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Order metrics
	OrdersCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Total number of orders created",
		},
	)

	OrdersOutOfStock = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_out_of_stock_total",
			Help: "Total number of orders that failed due to out of stock",
		},
	)

	// Job metrics
	JobsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_created_total",
			Help: "Total number of jobs created",
		},
		[]string{"type"},
	)

	JobsCompleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_completed_total",
			Help: "Total number of jobs completed",
		},
		[]string{"type", "status"},
	)

	JobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "job_duration_seconds",
			Help:    "Duration of job processing in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"type"},
	)

	// Database metrics
	DatabaseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation"},
	)
)

// Init initializes metrics (using promauto, metrics are auto-registered)
func Init() {
	// Metrics are automatically registered by promauto
}
