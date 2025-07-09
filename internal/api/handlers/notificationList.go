package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"notification/internal/monitoring"
	"notification/internal/storage/postgresClient"
)

func NewListNotificationHandler(pc postgresClient.PostgresClient, logger *zap.Logger, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//start := time.Now()

		handlerNameForMetrics := "ListNotification"

		if ctx.Err() != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncCanceled(handlerNameForMetrics)
			logger.Error("NewListNotificationHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		query := r.URL.Query()
		email, err := switchQuery(ctx, pc, query)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncError(handlerNameForMetrics)
			logger.Error("NewListNotificationHandler: cannot get email from postgres", zap.Error(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(email)
	}
}

//by=all-of-delayed
//by=all-of-instant

func switchQuery(ctx context.Context, pc postgresClient.PostgresClient, q url.Values) (any, error) {
	by := q.Get("by")
	id := q.Get("id")
	mail := q.Get("mail")

	var res any
	var err error

	switch by {
	case "id":
		res, err = pc.FetchById(ctx, id)
	case "mail":
		res, err = pc.FetchByMail(ctx, mail)
	case "all":
		//fetchByAll()
	}

	return res, err
}
