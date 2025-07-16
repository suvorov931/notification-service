package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
)

var ErrInvalidQuery = errors.New("invalid query")

func (nh *NotificationHandler) NewListNotificationHandler(metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), nh.calculateTimeoutForList())
		defer cancel()

		start := time.Now()

		handlerName := "ListNotification"

		if nh.checkCtxError(ctx, w, metrics, handlerName) {
			return
		}

		query := r.URL.Query()

		emails, err := nh.handleQuery(ctx, query)
		if err != nil {
			nh.processError(ctx, err, handlerName, metrics, w, r)
			return
		}

		nh.writeResponse(w, metrics, handlerName, emails)

		metrics.Observe(handlerName, start)
		metrics.IncSuccess(handlerName)
	}
}

func (nh *NotificationHandler) handleQuery(ctx context.Context, q url.Values) ([]*SMTPClient.EmailMessage, error) {
	by := q.Get("by")

	mail := q.Get("email")
	id := q.Get("id")

	switch by {
	case "id":
		intId, _ := strconv.Atoi(id)

		return nh.postgresClient.FetchById(ctx, intId)

	case "email":
		return nh.postgresClient.FetchByEmail(ctx, mail)

	case "all":
		return nh.postgresClient.FetchByAll(ctx)

	default:
		return nil, ErrInvalidQuery
	}
}

func (nh *NotificationHandler) writeResponse(w http.ResponseWriter, metrics monitoring.Monitoring, handlerName string, emails []*SMTPClient.EmailMessage) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(emails); err != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		metrics.IncError(handlerName)
		nh.logger.Error("NewListNotificationHandler: cannot send response to caller", zap.Error(err))
		return
	}
}

func (nh *NotificationHandler) processError(ctx context.Context, err error, handlerName string,
	metrics monitoring.Monitoring, w http.ResponseWriter, r *http.Request) {
	switch {
	case errors.Is(err, context.Canceled):
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		metrics.IncCanceled(handlerName)
		nh.logger.Info("NewListNotificationHandler: Context canceled after handleQuery", zap.Error(ctx.Err()))

		return

	case errors.Is(err, context.DeadlineExceeded):
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		metrics.IncTimeout(handlerName)
		nh.logger.Info("NewListNotificationHandler: Context deadline exceeded", zap.Error(ctx.Err()))

		return

	case errors.Is(err, pgx.ErrNoRows):
		http.Error(w, "There are no results for the specified param\n", http.StatusBadRequest)
		nh.logger.Warn("NewListNotificationHandler: no rows found", zap.Error(err), zap.String("query", r.URL.RawQuery))

		return

	case errors.Is(err, ErrInvalidQuery):
		http.Error(w, err.Error(), http.StatusBadRequest)
		nh.logger.Warn("NewListNotificationHandler: invalid query", zap.Error(err), zap.String("query", r.URL.RawQuery))

	default:
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		metrics.IncError(handlerName)
		nh.logger.Error("NewListNotificationHandler: cannot get email from postgres", zap.Error(err))

		return
	}
}
