package api

const (
	KeyForInstantSending = "instantSending"
	KeyForDelayedSending = "delayedSending"
)

type HttpServer struct {
	Host string `yaml:"HTTP_HOST" env:"HTTP_HOST"`
	Port string `yaml:"HTTP_PORT" env:"HTTP_PORT"`
}
