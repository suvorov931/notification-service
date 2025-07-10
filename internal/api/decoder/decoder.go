package decoder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"go.uber.org/zap"

	"notification/internal/SMTPClient"
)

const emailTimeLayout = "2006-01-02 15:04:05"

var (
	ErrNotAllFields            = errors.New("checkFields: request body not all required fields are filled")
	ErrNoValidRecipientAddress = errors.New("checkFields: no valid recipient address found")
	ErrHeaderNotJSON           = errors.New("checkHeaders: header is not a application/json")
	ErrSyntaxError             = errors.New("errDuringParse: request body contains badly-formed JSON")
	ErrInvalidType             = errors.New("errDuringParse: request body contains an invalid value type")
	ErrEmptyBody               = errors.New("decodeBody: request body must not be empty")
	ErrTimeNotAtFuture         = errors.New("checkTime: time not at future")
	ErrNoValidTimeFiled        = errors.New("checkTime: no valid time field")
	ErrUnknownError            = errors.New("unknown error")
)

type Decoder[T any] struct {
	logger *zap.Logger
	r      *http.Request
	w      http.ResponseWriter
}

func NewDecoder[T any](logger *zap.Logger, r *http.Request, w http.ResponseWriter) Decoder[T] {
	return Decoder[T]{
		logger: logger,
		r:      r,
		w:      w,
	}
}

func (d Decoder[T]) Decode() (*T, error) {
	var email T

	if err := d.checkHeaders(); err != nil {
		return nil, err
	}

	if err := d.decodeBody(&email); err != nil {
		return nil, d.errDuringParse(err)
	}

	if err := d.checkFields(&email); err != nil {
		return nil, err
	}

	return &email, nil
}

func (d Decoder[T]) checkHeaders() error {
	ct := d.r.Header.Get("Content-Type")
	if ct != "application/json" {
		d.logger.Error(ErrHeaderNotJSON.Error())
		http.Error(d.w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)

		return ErrHeaderNotJSON
	}

	return nil
}

func (d Decoder[T]) decodeBody(target *T) error {
	bodyBytes, err := io.ReadAll(d.r.Body)
	if err != nil {
		d.logger.Error("decodeBody: failed to read request body", zap.Error(err))
		http.Error(d.w, "Failed to read request body", http.StatusInternalServerError)
		return ErrUnknownError
	}
	defer d.r.Body.Close()

	if len(bodyBytes) == 0 {
		d.logger.Error(ErrEmptyBody.Error())
		http.Error(d.w, "Request body must not be empty", http.StatusBadRequest)
		return ErrEmptyBody
	}

	d.r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	dec := json.NewDecoder(d.r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}

func (d Decoder[T]) errDuringParse(err error) error {
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

func (d Decoder[T]) checkFields(email *T) error {
	switch v := any(*email).(type) {
	case SMTPClient.EmailMessage:
		if err := d.validateCommonFields(v.To, v.Subject, v.Message); err != nil {
			return err
		}

	case SMTPClient.EmailMessageWithTime:
		if err := d.validateCommonFields(v.To, v.Subject, v.Message); err != nil {
			return err
		}

		if v.Time == "" {
			d.logger.Error(ErrNotAllFields.Error())
			http.Error(d.w, "Not all fields in the request body are filled in", http.StatusBadRequest)
			return ErrNotAllFields
		}

		if err := d.checkTime(v.Time); err != nil {
			return err
		}

	default:
		return ErrUnknownError
	}

	return nil
}

func (d Decoder[T]) validateCommonFields(to, subject, message string) error {
	if to == "" || subject == "" || message == "" {
		d.logger.Error(ErrNotAllFields.Error())
		http.Error(d.w, "Not all fields in the request body are filled in", http.StatusBadRequest)
		return ErrNotAllFields
	}

	if _, err := mail.ParseAddress(to); err != nil {
		d.logger.Error(ErrNoValidRecipientAddress.Error())
		http.Error(d.w, "No valid recipient address found", http.StatusBadRequest)
		return ErrNoValidRecipientAddress
	}

	return nil
}

func (d Decoder[T]) checkTime(t string) error {
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
