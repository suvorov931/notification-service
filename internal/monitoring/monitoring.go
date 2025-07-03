package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	StatusSuccess  = "success"
	StatusError    = "error"
	StatusCanceled = "canceled"
	StatusTimeout  = "timeout"
)

type Monitoring interface {
	Inc(operation string, status string)
	Observe(operation string, duration float64)
}

type Metrics struct {
	Counter  *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

type AppMetrics struct {
	RedisMetrics                   *Metrics
	WorkerMetrics                  *Metrics
	SMTPMetrics                    *Metrics
	SendNotificationMetrics        *Metrics
	SendNotificationViaTimeMetrics *Metrics
}

func NewAppMetrics() *AppMetrics {
	return &AppMetrics{
		RedisMetrics:                   New("Redis"),
		WorkerMetrics:                  New("Worker"),
		SMTPMetrics:                    New("SMTP"),
		SendNotificationMetrics:        New("SendNotification"),
		SendNotificationViaTimeMetrics: New("SendNotificationViaTime"),
	}
}

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

func (m *Metrics) Inc(operation string, status string) {
	m.Counter.WithLabelValues(operation, status).Inc()
}

func (m *Metrics) Observe(operation string, duration float64) {
	m.Duration.WithLabelValues(operation).Observe(duration)
}

type NopMetrics struct{}

func NewNop() *NopMetrics {
	return &NopMetrics{}
}

func (nm *NopMetrics) Inc(operation string, status string)        {}
func (nm *NopMetrics) Observe(operation string, duration float64) {}
