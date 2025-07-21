package api

import "time"

const (
	// KeyForInstantSending is a constant key used across the project to indicate instant email sending.
	KeyForInstantSending = "instantSending"

	// KeyForDelayedSending is a constant key used across the project to indicate delayed email sending.
	KeyForDelayedSending = "delayedSending"
)

// HttpServer defines the configuration parameters for the HTTP server.
type HttpServer struct {
	Host           string        `env:"HTTP_HOST"`
	Port           string        `env:"HTTP_PORT"`
	MonitoringPort string        `env:"HTTP_MONITORING_PORT"`
	TimeoutExtra   time.Duration `env:"HTTP_TIMEOUT_EXTRA"`
}
