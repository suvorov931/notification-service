package api

const (
	KeyForInstantSending = "instantSending"
	KeyForDelayedSending = "delayedSending"
)

type HttpServer struct {
	Host           string `env:"HTTP_HOST"`
	Port           string `env:"HTTP_PORT"`
	MonitoringPort string `env:"HTTP_MONITORING_PORT"`
}
