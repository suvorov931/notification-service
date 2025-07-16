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

const extraTimeForTimeout = 3 * time.Second

type NotificationHandler struct {
	logger         *zap.Logger
	sender         SMTPClient.EmailSender
	redisClient    redisClient.RedisClient
	postgresClient postgresClient.PostgresClient
	timeouts       config.AppTimeouts
}

func New(logger *zap.Logger, sender SMTPClient.EmailSender, redisClient redisClient.RedisClient,
	postgresClient postgresClient.PostgresClient, timeouts config.AppTimeouts) *NotificationHandler {
	return &NotificationHandler{
		logger:         logger,
		sender:         sender,
		redisClient:    redisClient,
		postgresClient: postgresClient,
		timeouts:       timeouts,
	}
}

func (nh *NotificationHandler) calculateTimeoutForSend() time.Duration {
	var smtpAllTimeout time.Duration

	for i := 0; i < nh.timeouts.SMTPQuantityOfRetries+1; i++ {
		smtpAllTimeout += nh.sender.CreatePause(i)
	}

	allTimeout := smtpAllTimeout + nh.timeouts.PostgresTimeout + extraTimeForTimeout

	return allTimeout
}

func (nh *NotificationHandler) calculateTimeoutForSendViaTime() time.Duration {
	allTimeout := nh.timeouts.RedisTimeout + nh.timeouts.PostgresTimeout + extraTimeForTimeout
	return allTimeout
}

func (nh *NotificationHandler) calculateTimeoutForList() time.Duration {
	allTimeout := nh.timeouts.PostgresTimeout + extraTimeForTimeout
	return allTimeout
}

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

type respMessage struct {
	Message string `json:"message"`
	Id      int    `json:"id,omitempty"`
}

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
