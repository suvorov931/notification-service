package decoder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"time"

	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/api"
)

const emailTimeLayout = "2006-01-02 15:04:05"

var (
	ErrNotAllFields            = errors.New("DecodeMailRequest: checkFields: request body not all required fields are filled")
	ErrNoValidRecipientAddress = errors.New("DecodeMailRequest: checkFields: no valid recipient address found")
	ErrHeaderNotJSON           = errors.New("DecodeMailRequest: checkHeaders: header is not a application/json")
	ErrSyntaxError             = errors.New("DecodeMailRequest: errDuringParse: request body contains badly-formed JSON")
	ErrInvalidType             = errors.New("DecodeMailRequest: errDuringParse: request body contains an invalid value type")
	ErrEmptyBody               = errors.New("DecodeMailRequest: decodeBody: request body must not be empty")
	ErrTimeNotAtFuture         = errors.New("DecodeMailRequest: checkTime: time not at future")
	ErrNoValidTimeFiled        = errors.New("DecodeMailRequest: checkTime: no valid time field")
	ErrUnknownKey              = errors.New("DecodeMailRequest: determineType: unknown key")
	ErrUnknownError            = errors.New("DecodeMailRequest: Unknown error")
)

// TODO: попробовать переделать через дженерик

type decoder struct {
	logger *zap.Logger
	r      *http.Request
	w      http.ResponseWriter
}

func DecodeEmailRequest(key string, w http.ResponseWriter, r *http.Request, logger *zap.Logger) (any, error) {
	d := &decoder{
		logger: logger,
		r:      r,
		w:      w,
	}

	if err := d.checkHeaders(); err != nil {
		return nil, err
	}

	email, err := d.createEmailModel(key)
	if err != nil {
		return nil, err
	}

	if err = d.decodeBody(email); err != nil {
		return nil, d.errDuringParse(err)
	}

	return d.checkFields(email)
}

func (d *decoder) checkHeaders() error {
	ct := d.r.Header.Get("Content-Type")
	if ct != "application/json" {
		d.logger.Error(ErrHeaderNotJSON.Error())
		http.Error(d.w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)

		return ErrHeaderNotJSON
	}

	return nil
}

func (d *decoder) createEmailModel(key string) (any, error) {
	var email any
	switch key {
	case api.KeyForInstantSending:
		email = &SMTPClient.EmailMessage{}
	case api.KeyForDelayedSending:
		email = &SMTPClient.TempEmailMessageWithTime{}
	default:
		http.Error(d.w, http.StatusText(500), http.StatusInternalServerError)

		d.logger.Error(ErrUnknownKey.Error(), zap.String("key", key))
		return nil, ErrUnknownKey
	}

	return email, nil
}

func (d *decoder) decodeBody(email any) error {
	body, err := io.ReadAll(d.r.Body)
	if err != nil {
		d.logger.Error(ErrEmptyBody.Error())
		http.Error(d.w, "Failed to read request body", http.StatusBadRequest)

		return ErrEmptyBody
	}

	if len(body) == 0 {
		d.logger.Error(ErrEmptyBody.Error())
		http.Error(d.w, "Request body must not be empty", http.StatusBadRequest)

		return ErrEmptyBody
	}

	d.r.Body = io.NopCloser(bytes.NewBuffer(body))

	dec := json.NewDecoder(d.r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(email)
}

func (d *decoder) errDuringParse(err error) error {
	if errors.Is(err, ErrEmptyBody) {
		return ErrEmptyBody
	}

	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError

	switch {
	case errors.As(err, &syntaxError):
		d.logger.Error(ErrSyntaxError.Error())
		http.Error(d.w,
			fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset),
			http.StatusBadRequest)

		return ErrSyntaxError

	case errors.As(err, &unmarshalTypeError):
		d.logger.Error(ErrInvalidType.Error())
		http.Error(d.w,
			fmt.Sprintf(
				"Request body contains an invalid value for the %q field (at position %d)",
				unmarshalTypeError.Field, unmarshalTypeError.Offset),
			http.StatusBadRequest)

		return ErrInvalidType

	default:
		d.logger.Error(ErrUnknownError.Error())
		http.Error(d.w, http.StatusText(500), http.StatusInternalServerError)

		return ErrUnknownError
	}
}

func (d *decoder) checkFields(email any) (any, error) {
	switch t := email.(type) {
	case *SMTPClient.EmailMessage:
		if t.To == "" || t.Message == "" || t.Subject == "" {
			d.logger.Error(ErrNotAllFields.Error())
			http.Error(d.w, "Not all fields in the request body are filled in", http.StatusBadRequest)

			return nil, ErrNotAllFields
		}

		if _, err := mail.ParseAddress(t.To); err != nil {
			d.logger.Error(ErrNoValidRecipientAddress.Error())
			http.Error(d.w, "No valid recipient address found", http.StatusBadRequest)

			return nil, ErrNoValidRecipientAddress
		}

		return email, nil

	case *SMTPClient.TempEmailMessageWithTime:
		if t.Time == "" || t.To == "" || t.Message == "" || t.Subject == "" {
			d.logger.Error(ErrNotAllFields.Error())
			http.Error(d.w, "Not all fields in the request body are filled in", http.StatusBadRequest)

			return nil, ErrNotAllFields
		}

		if err := d.checkTime(t.Time); err != nil {
			return nil, err
		}

		if _, err := mail.ParseAddress(t.To); err != nil {
			d.logger.Error(ErrNoValidRecipientAddress.Error())
			http.Error(d.w, "No valid recipient address found", http.StatusBadRequest)

			return nil, ErrNoValidRecipientAddress
		}

		res := SMTPClient.EmailMessageWithTime{
			Time: t.Time,
			Email: SMTPClient.EmailMessage{
				To:      t.To,
				Subject: t.Subject,
				Message: t.Message,
			},
		}

		return &res, nil
	}

	return nil, ErrUnknownError
}

func (d *decoder) checkTime(t string) error {
	UTCTime, err := time.ParseInLocation(emailTimeLayout, t, time.UTC)
	if err != nil {
		d.logger.Info(ErrNoValidTimeFiled.Error())
		http.Error(d.w, "The specified time is not a valid", http.StatusBadRequest)

		return ErrNoValidTimeFiled
	}

	if !UTCTime.After(time.Now()) {
		d.logger.Info(ErrTimeNotAtFuture.Error())
		http.Error(d.w, "The specified time is not in the future", http.StatusBadRequest)

		return ErrTimeNotAtFuture
	}

	return nil
}
