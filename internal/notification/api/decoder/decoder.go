package decoder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	"notification/internal/notification/service"
)

func DecodeMailRequest(w http.ResponseWriter, r *http.Request, l *zap.Logger) (*service.Mail, error) {
	ct := r.Header.Get("Content-Type")
	if ct != "application/json" {
		l.Error(fmt.Sprintf("DecodeMailRequest: header: %s is not a application/json", r.Header))
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return nil, fmt.Errorf(`Content-Type must be "application/json" or "application/json"`)
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var mail service.Mail

	err := dec.Decode(&mail)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			l.Error("DecodeMailRequest:" + (msg))
			http.Error(w, msg, http.StatusBadRequest)

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf(
				"Request body contains an invalid value for the %q field (at position %d)",
				unmarshalTypeError.Field,
				unmarshalTypeError.Offset,
			)
			l.Error("DecodeMailRequest:" + msg)
			http.Error(w, msg, http.StatusBadRequest)

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			l.Error("DecodeMailRequest:" + msg)
			http.Error(w, msg, http.StatusBadRequest)

		default:
			l.Error("DecodeMailRequest: unknown error", zap.Error(err))
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		}
	}

	return &mail, nil
}
