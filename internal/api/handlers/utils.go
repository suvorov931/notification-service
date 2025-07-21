package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/config"
	"notification/internal/monitoring"
	"notification/internal/storage/postgresClient"
	"notification/internal/storage/redisClient"
)

// NotificationHandler handles email notification HTTP requests.
// It manages timeout configurations used for request processing.
type NotificationHandler struct {
	logger         *zap.Logger
	sender         SMTPClient.EmailSender
	redisClient    redisClient.RedisClient
	postgresClient postgresClient.PostgresClient
	timeouts       config.AppTimeouts
	extraTimeout   time.Duration
}

// New creates and returns a new NotificationHandler instance.
func New(logger *zap.Logger, sender SMTPClient.EmailSender, redisClient redisClient.RedisClient,
	postgresClient postgresClient.PostgresClient, timeouts config.AppTimeouts, extraTimeout time.Duration) *NotificationHandler {
	return &NotificationHandler{
		logger:         logger,
		sender:         sender,
		redisClient:    redisClient,
		postgresClient: postgresClient,
		timeouts:       timeouts,
		extraTimeout:   extraTimeout,
	}
}

// calculateTimeoutForSend calculates the total timeout for NewSendNotificationHandler,
// including SMTP retry delays, PostgreSQL timeout, and additional buffer time.
func (nh *NotificationHandler) calculateTimeoutForSend() time.Duration {
	var smtpAllTimeout time.Duration

	for i := 0; i < nh.timeouts.SMTPQuantityOfRetries+1; i++ {
		smtpAllTimeout += nh.sender.CreatePause(i)
	}

	allTimeout := smtpAllTimeout + nh.timeouts.PostgresTimeout + nh.extraTimeout

	return allTimeout
}

// calculateTimeoutForSend calculates the total timeout for NewSendNotificationViaTimeHandler,
// including Redis timeout, PostgreSQL timeout, and additional buffer time.
func (nh *NotificationHandler) calculateTimeoutForSendViaTime() time.Duration {
	allTimeout := nh.timeouts.RedisTimeout + nh.timeouts.PostgresTimeout + nh.extraTimeout
	return allTimeout
}

// calculateTimeoutForSend calculates the total timeout for NewListNotificationHandler,
// including PostgreSQL timeout, and additional buffer time.
func (nh *NotificationHandler) calculateTimeoutForList() time.Duration {
	allTimeout := nh.timeouts.PostgresTimeout + nh.extraTimeout
	return allTimeout
}

// checkCtxError checks which one exactly context error (context canceled or deadline exceeded).
func (nh *NotificationHandler) checkCtxError(ctx context.Context, w http.ResponseWriter,
	metrics monitoring.Monitoring, handlerName string) bool {

	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		metrics.IncCanceled(handlerName)
		nh.logger.Error(handlerName+": Context canceled before processing started", zap.Error(ctx.Err()))

		return true

	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		metrics.IncCanceled(handlerName)
		nh.logger.Error(handlerName+": Context deadline before processing started", zap.Error(ctx.Err()))

		return true

	default:
		return false
	}
}

// respMessage is an auxiliary structure for writeResponseWithId.
type respMessage struct {
	Message string `json:"message"`
	Id      int    `json:"id"`
}

// writeResponseWithId writes a JSON response for the HTTP client containing the specified id number from PostgreSQL.
func (nh *NotificationHandler) writeResponseWithId(w http.ResponseWriter, id int, message string, metrics monitoring.Monitoring, handlerName string) {
	resp := respMessage{
		Message: message,
		Id:      id,
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		metrics.IncError(handlerName)
		nh.logger.Error(handlerName+": Cannot send report to caller", zap.Error(err))
	}
}
