package logger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Env string
}

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

		logger, err := config.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}

		return logger, nil
	default:
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("can't initialize logger: %w", err)
		}
		return logger, nil
	}
}

func MiddlewareLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := logger.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				//zap.String("remote_addr", r.RemoteAddr),
				//zap.String("user_agent", r.UserAgent()),
				//zap.String("request_id", middleware.GetReqID(r.Context())),
				//zap.Time("time", time.Now()),
			)

			//start := time.Now()
			entry.Info("new request")
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				entry.Info(
					"request completed",
					zap.Int("status", ww.Status()),
					//zap.Duration("duration", time.Since(start)),
				)
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
