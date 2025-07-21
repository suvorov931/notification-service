package logger

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		errMsg string
	}{
		{
			name:   "development environment",
			env:    "dev",
			errMsg: "",
		},
		{
			name:   "production environment",
			env:    "prod",
			errMsg: "",
		},
		{
			name:   "empty environment",
			env:    "",
			errMsg: "unknown environment",
		},
		{
			name:   "invalid environment",
			env:    "foo",
			errMsg: "unknown environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			logger, err := New(cfg)

			if tt.errMsg != "" {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errMsg)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestMiddlewareLogger(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{
			name: "development environment",
			env:  "dev",
		},
		{
			name: "production environment",
			env:  "prod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := &Config{
				Env:    "prod",
				output: &buf,
			}
			logger, err := New(cfg)

			assert.NoError(t, err)
			assert.NotNil(t, logger)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err = w.Write([]byte("OK")); err != nil {
					logger.Warn("cannot write response", zap.Error(err))
				}
			})

			middlewareHandler := MiddlewareLogger(logger, cfg)(nextHandler)

			req := httptest.NewRequest("GET", "/test-path", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			req.Header.Set("User-Agent", "GoTest")
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-id"))

			rec := httptest.NewRecorder()

			middlewareHandler.ServeHTTP(rec, req)

			logOutput := buf.String()

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "OK", rec.Body.String())

			assert.Contains(t, logOutput, "new request")
			assert.Contains(t, logOutput, "request completed")
			assert.Contains(t, logOutput, "/test-path")
			assert.Contains(t, logOutput, "GET")
			assert.Contains(t, logOutput, "request_id")
			assert.Contains(t, logOutput, "test-request-id")
		})
	}
}
