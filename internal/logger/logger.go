package logger

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config defines the logger settings, (field "output" is optional and intended for tests).
type Config struct {
	Env    string `yaml:"ENV" env:"LOGGER"`
	output io.Writer
}

// New creates and returns a new logger instance with specified parameters.
// In "dev" mode, it uses a comfortable console encoder for local development.
// In prod mode (and others), it uses a JSON encoder suitable for structured logging in production.
func New(cfg *Config) (*zap.Logger, error) {
	switch cfg.Env {
	case "dev":
		config := zap.NewDevelopmentConfig()

		config.DisableCaller = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.LineEnding = "\n\n"
		config.EncoderConfig.ConsoleSeparator = " | "
		config.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("\033[36m" + t.Format("15:04:05") + "\033[0m")
		}

		if cfg.output != nil {
			config.OutputPaths = []string{"stdout"}
			core := zapcore.NewCore(
				zapcore.NewConsoleEncoder(config.EncoderConfig),
				zapcore.AddSync(cfg.output),
				config.Level,
			)

			logger := zap.New(core)

			return logger, nil

		} else {
			logger, err := config.Build()
			if err != nil {
				return nil, err
			}

			return logger, nil

		}

	case "prod":
		if cfg.output != nil {
			config := zap.NewProductionConfig()
			core := zapcore.NewCore(
				zapcore.NewJSONEncoder(config.EncoderConfig),
				zapcore.AddSync(cfg.output),
				config.Level,
			)

			logger := zap.New(core)

			return logger, nil

		} else {
			logger, err := zap.NewProduction()
			if err != nil {
				return nil, err
			}

			return logger, nil
		}

	default:
		return nil, fmt.Errorf("unknown environment: %s", cfg.Env)
	}
}

// MiddlewareLogger creates an HTTP middleware that logs details about incoming HTTP request.
// In "dev" mode, it logs basic information: HTTP method, path and response status.
// In prod mode (and others), it logs extended details such as:
// method, path, remote address, user agent, request id, processing time and response status.
func MiddlewareLogger(logger *zap.Logger, cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := logger.With()
			start := time.Now()

			switch cfg.Env {
			case "dev":
				entry = logger.With(
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)

				entry.Info("new request")

			default:
				entry = logger.With(
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.String("request_id", middleware.GetReqID(r.Context())),
					zap.Time("time", time.Now()),
				)

				entry.Info("new request")
			}
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				switch cfg.Env {
				case "dev":
					entry.Info(
						"request completed",
						zap.Int("status", ww.Status()),
					)

				default:
					entry.Info(
						"request completed",
						zap.Int("status", ww.Status()),
						zap.Duration("duration", time.Since(start)),
					)
				}
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
