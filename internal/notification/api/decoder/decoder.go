package decoder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"

	"go.uber.org/zap"

	"notification/internal/notification/service"
)

var (
	ErrNotAllFields            = errors.New("DecodeMailRequest: Request body not all required fields are filled")
	ErrHeaderNotJSON           = errors.New("DecodeMailRequest: Header is not a application/json")
	ErrSyntaxError             = errors.New("DecodeMailRequest: Request body contains badly-formed JSON")
	ErrInvalidType             = errors.New("DecodeMailRequest: Request body contains an invalid value type")
	ErrEmptyBody               = errors.New("DecodeMailRequest: Request body must not be empty")
	ErrNoValidRecipientAddress = errors.New("DecodeMailRequest: No valid recipient address found")
	ErrUnknownError            = errors.New("DecodeMailRequest: Unknown error")
)

func DecodeMailRequest(w http.ResponseWriter, r *http.Request, l *zap.Logger) (*service.Email, error) {
	ct := r.Header.Get("Content-Type")
	if ct != "application/json" {
		l.Error(ErrHeaderNotJSON.Error())
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)

		return nil, ErrHeaderNotJSON
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		l.Error(ErrEmptyBody.Error())
		http.Error(w, "Failed to read request body", http.StatusBadRequest)

		return nil, ErrEmptyBody
	}
	if len(body) == 0 {
		l.Error(ErrEmptyBody.Error())
		http.Error(w, "Request body must not be empty", http.StatusBadRequest)

		return nil, ErrEmptyBody
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	var email service.Email

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err = dec.Decode(&email)

	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			l.Error(ErrSyntaxError.Error())
			http.Error(w,
				fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset),
				http.StatusBadRequest)

			return nil, ErrSyntaxError

		case errors.As(err, &unmarshalTypeError):
			l.Error(ErrInvalidType.Error())
			http.Error(w,
				fmt.Sprintf(
					"Request body contains an invalid value for the %q field (at position %d)",
					unmarshalTypeError.Field, unmarshalTypeError.Offset),
				http.StatusBadRequest)

			return nil, ErrInvalidType

		default:
			l.Error(ErrUnknownError.Error())
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)

			return nil, ErrUnknownError
		}
	}
	if _, err = mail.ParseAddress(email.To); err != nil {
		l.Error(ErrNoValidRecipientAddress.Error())
		http.Error(w, "No valid recipient address found", http.StatusBadRequest)
		return nil, ErrNoValidRecipientAddress
	}

	if email.To == "" || email.Message == "" || email.Subject == "" {
		l.Error(ErrNotAllFields.Error())
		http.Error(w, "Not all fields in the request body are filled in", http.StatusBadRequest)

		return nil, ErrNotAllFields
	}

	return &email, nil
}
