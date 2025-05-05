package handler

import (
	"net/http"

	"go.uber.org/zap"

	"notification/internal/notification/api/decoder"
	"notification/internal/notification/service"
)

func NewSendNotificationHandler(l *zap.Logger, sender service.MailSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mail, err := decoder.DecodeMailRequest(w, r, l)
		if err != nil {
			return
		}

		_, err = w.Write([]byte("Message is correct,\nStarting to send notification\n\n"))
		if err != nil {
			l.Warn("cannot send report to caller", zap.Error(err))
		}

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
