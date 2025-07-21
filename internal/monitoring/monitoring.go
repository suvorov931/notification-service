package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	StatusSuccess  = "success"
	StatusError    = "error"
	StatusCanceled = "canceled"
	StatusTimeout  = "timeout"
)

// Monitoring defines an interface for recording operational metrics such as,
// counters by status and execution durations.
type Monitoring interface {
	IncSuccess(operation string)
	IncError(operation string)
	IncCanceled(operation string)
	IncTimeout(operation string)
	Observe(operation string, start time.Time)
}

// Metrics implements a Monitoring interface,
// contains the counting operations and observing execution durations.
type Metrics struct {
	Counter  *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

// AppMetrics contains the named metrics group for different components in notification-service.
type AppMetrics struct {
	RedisMetrics                   *Metrics
	PostgresMetrics                *Metrics
	WorkerMetrics                  *Metrics
	SMTPMetrics                    *Metrics
	ListNotificationMetrics        *Metrics
	SendNotificationMetrics        *Metrics
	SendNotificationViaTimeMetrics *Metrics
}

// NewAppMetrics creates and returns a new AppMetrics instance.
func NewAppMetrics() *AppMetrics {
	return &AppMetrics{
		RedisMetrics:                   New("Redis"),
		PostgresMetrics:                New("Postgres"),
		WorkerMetrics:                  New("Worker"),
		SMTPMetrics:                    New("SMTP"),
		ListNotificationMetrics:        New("ListNotification"),
		SendNotificationMetrics:        New("SendNotification"),
		SendNotificationViaTimeMetrics: New("SendNotificationViaTime"),
	}
}

// New creates and returns a new named Metrics instance, includes the counter and histogram for operation time duration.
func New(name string) *Metrics {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name + "_operations_total",
		Help: "Total count of " + name + " operations",
	},
		[]string{"operation", "status"})

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name + "_operation_duration_seconds",
		Help:    "Duration of " + name + " operations",
		Buckets: prometheus.DefBuckets,
	},
		[]string{"operation"})

	prometheus.MustRegister(counter, duration)

	return &Metrics{
		Counter:  counter,
		Duration: duration,
	}
}

// IncSuccess increments the success status for the specified name operation.
func (m *Metrics) IncSuccess(operation string) {
	m.Counter.WithLabelValues(operation, StatusSuccess).Inc()
}

// IncError increments the error status for the specified name operation.
func (m *Metrics) IncError(operation string) {
	m.Counter.WithLabelValues(operation, StatusError).Inc()
}

// IncCanceled increments the canceled status for the specified name operation.
func (m *Metrics) IncCanceled(operation string) {
	m.Counter.WithLabelValues(operation, StatusCanceled).Inc()
}

// IncTimeout increments the timeout status for the specified name operation.
func (m *Metrics) IncTimeout(operation string) {
	m.Counter.WithLabelValues(operation, StatusTimeout).Inc()
}

// Observe records the execution time duration using specified start time for the specified name operation.
func (m *Metrics) Observe(operation string, start time.Time) {
	duration := time.Since(start).Seconds()
	m.Duration.WithLabelValues(operation).Observe(duration)
}

// NopMetrics is a no-op implementation of the Monitoring interface.
type NopMetrics struct{}

// NewNop create and return new NopMetrics instance.
func NewNop() *NopMetrics {
	return &NopMetrics{}
}

// IncSuccess is a no-op implementation.
func (nm *NopMetrics) IncSuccess(operation string) {}

// IncError is a no-op implementation.
func (nm *NopMetrics) IncError(operation string) {}

// IncCanceled is a no-op implementation.
func (nm *NopMetrics) IncCanceled(operation string) {}

// IncTimeout is a no-op implementation.
func (nm *NopMetrics) IncTimeout(operation string) {}

// Observe is a no-op implementation.
func (nm *NopMetrics) Observe(operation string, start time.Time) {}
