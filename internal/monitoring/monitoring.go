package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RedisSuccessCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "redis_success_total",
		Help: "Total number of successful Redis operations",
	})

	RedisErrorCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "redis_error_total",
		Help: "Total number of error Redis operations",
	})

	RedisLatencyHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "redis_operation_duration_seconds",
		Help:    "Duration of Redis operations in seconds",
		Buckets: prometheus.DefBuckets,
	})
)
