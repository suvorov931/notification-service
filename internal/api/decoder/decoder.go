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
	"notification/internal/api"
)

const emailTimeLayout = "2006-01-02 15:04:05"

var (
	errNotAllFields            = errors.New("checkFields: request body not all required fields are filled")
	errNoValidRecipientAddress = errors.New("checkFields: no valid recipient address found")
	errHeaderNotJSON           = errors.New("checkHeaders: header is not a application/json")
	errSyntaxError             = errors.New("errDuringParse: request body contains badly-formed JSON")
	errInvalidType             = errors.New("errDuringParse: request body contains an invalid value type")
	errEmptyBody               = errors.New("decodeBody: request body must not be empty")
	errTimeNotAtFuture         = errors.New("checkTime: time not at future")
	errNoValidTimeField        = errors.New("checkTime: no valid time field")
	errUnknownError            = errors.New("unknown error")
)

type decoder struct {
	logger *zap.Logger
	r      *http.Request
	w      http.ResponseWriter
}

func DecodeRequest(logger *zap.Logger, r *http.Request, w http.ResponseWriter, sendingType string) (*SMTPClient.EmailMessage, error) {
	d := decoder{
		logger: logger,
		r:      r,
		w:      w,
	}

	if err := d.checkHeaders(); err != nil {
		return nil, err
	}

	email := &SMTPClient.EmailMessage{}

	if err := d.decodeBody(email); err != nil {
		return nil, d.errDuringParse(err)
	}

	return d.checkFields(email, sendingType)
}

func (d *decoder) checkHeaders() error {
	ct := d.r.Header.Get("Content-Type")
	if ct != "application/json" {
		d.logger.Error(errHeaderNotJSON.Error())
		http.Error(d.w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)

		return errHeaderNotJSON
	}

	return nil
}

func (d *decoder) decodeBody(email *SMTPClient.EmailMessage) error {
	bodyBytes, err := io.ReadAll(d.r.Body)
	if err != nil {
		d.logger.Error("decodeBody: failed to read request body", zap.Error(err))
		http.Error(d.w, "Failed to read request body", http.StatusInternalServerError)
		return errUnknownError
	}
	defer d.r.Body.Close()

	if len(bodyBytes) == 0 {
		d.logger.Error(errEmptyBody.Error())
		http.Error(d.w, "Request body must not be empty", http.StatusBadRequest)
		return errEmptyBody
	}

	d.r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	dec := json.NewDecoder(d.r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(email)
}

func (d *decoder) errDuringParse(err error) error {
	if errors.Is(err, errEmptyBody) {
		return errEmptyBody
	}

	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError

	switch {
	case errors.As(err, &syntaxError):
		d.logger.Error(errSyntaxError.Error())
		http.Error(d.w,
			fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset),
			http.StatusBadRequest)

		return errSyntaxError

	case errors.As(err, &unmarshalTypeError):
		d.logger.Error(errInvalidType.Error())
		http.Error(d.w,
			fmt.Sprintf(
				"Request body contains an invalid value for the %q field (at position %d)",
				unmarshalTypeError.Field, unmarshalTypeError.Offset),
			http.StatusBadRequest)

		return errInvalidType

	default:
		d.logger.Error(errUnknownError.Error())
		http.Error(d.w, http.StatusText(500), http.StatusInternalServerError)

		return errUnknownError
	}
}

func (d *decoder) checkFields(email *SMTPClient.EmailMessage, sendingType string) (*SMTPClient.EmailMessage, error) {
	if email.To == "" || email.Subject == "" || email.Message == "" {
		d.logger.Error(errNotAllFields.Error())
		http.Error(d.w, "Not all fields in the request body are filled in", http.StatusBadRequest)
		return nil, errNotAllFields
	}

	if _, err := mail.ParseAddress(email.To); err != nil {
		d.logger.Error(errNoValidRecipientAddress.Error())
		http.Error(d.w, "No valid recipient address found", http.StatusBadRequest)
		return nil, errNoValidRecipientAddress
	}

	if sendingType == api.KeyForDelayedSending {
		err := d.checkTime(email.Time)
		if err != nil {
			return nil, err
		}
	}

	email.Type = sendingType

	return email, nil
}

func (d *decoder) checkTime(t string) error {
	UTCTime, err := time.ParseInLocation(emailTimeLayout, t, time.UTC)
	if err != nil {
		d.logger.Info(errNoValidTimeField.Error())
		http.Error(d.w, "The specified time is not a valid", http.StatusBadRequest)

		return errNoValidTimeField
	}

	if !UTCTime.After(time.Now()) {
		d.logger.Info(errTimeNotAtFuture.Error())
		http.Error(d.w, "The specified time is not in the future", http.StatusBadRequest)

		return errTimeNotAtFuture
	}

	return nil
}
