package handler

import (
	"net/http"

	"go.uber.org/zap"

	"notification/internal/notification/api"
	"notification/internal/notification/mail"
)

func NewSendNotificationHandler(l *zap.Logger, sender mail.MailSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mail, err := api.DecodeMailRequest(w, r, l)
		if err != nil {
			return
		}

		w.Write([]byte("Message is correct,\nStarting to send notification\n\n"))

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		} else {
			l.Warn("ResponseWriter does not support flushing")
		}

		if err = sender.SendMessage(*mail); err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			l.Error("SendNotification: cannot send notification", zap.Error(err))
			return
		}
		w.Write([]byte("Successfully sent notification\n"))
	}
}
