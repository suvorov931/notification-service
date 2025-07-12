package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
	"notification/internal/storage/postgresClient"
	"notification/internal/storage/redisClient"
)

type NotificationHandler struct {
	logger         *zap.Logger
	sender         SMTPClient.EmailSender
	redisClient    redisClient.RedisClient
	postgresClient postgresClient.PostgresClient
}

func New(logger *zap.Logger, sender SMTPClient.EmailSender,
	redisClient redisClient.RedisClient, postgresClient postgresClient.PostgresClient) *NotificationHandler {
	return &NotificationHandler{
		logger:         logger,
		sender:         sender,
		redisClient:    redisClient,
		postgresClient: postgresClient,
	}
}

func (nh *NotificationHandler) checkCtxCanceled(ctx context.Context, w http.ResponseWriter, metrics monitoring.Monitoring,
	handlerName string) bool {
	if ctx.Err() != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		metrics.IncCanceled(handlerName)
		nh.logger.Error(handlerName+": Context canceled before processing started", zap.Error(ctx.Err()))
		return true
	}

	return false
}

type respMessage struct {
	Message string `json:"message"`
	Id      int    `json:"id,omitempty"`
}

func (nh *NotificationHandler) writeResponseWithId(w http.ResponseWriter, id int, message string, metrics monitoring.Monitoring,
	handlerName string) {
	resp := respMessage{
		Message: message,
		Id:      id,
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		metrics.IncError(handlerName)
		nh.logger.Error("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
	}
}
