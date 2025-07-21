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

// emailTimeLayout required a time layout for checkTime function.
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

// decoder handles decoding and validation of HTTP requests.
type decoder struct {
	logger *zap.Logger
	r      *http.Request
	w      http.ResponseWriter
}

// DecodeRequest parses and validates the incoming HTTP request body.
// It checks the headers, required fields, recipient email address, and an optional time field (if needed).
// On success, it returns a parsed EmailMessage struct.
// On failure, it returns the corresponding error and writes an error message to the HTTP client.
func DecodeRequest(logger *zap.Logger, r *http.Request, w http.ResponseWriter, sendingType string) (*SMTPClient.EmailMessage, error) {
	d := decoder{
		logger: logger,
		r:      r,
		w:      w,
	}

	if err := d.checkHeaders(); err != nil {
		return nil, err
	}

	email := &SMTPClient.TempEmailMessage{}

	if err := d.decodeBody(email); err != nil {
		return nil, d.errDuringParse(err)
	}

	email, err := d.checkFields(email, sendingType)
	if err != nil {
		return nil, err
	}

	return d.convert(email)
}

// checkHeaders validates the Content-Type header and ensures it is set to application/json.
func (d *decoder) checkHeaders() error {
	ct := d.r.Header.Get("Content-Type")
	if ct != "application/json" {
		d.logger.Error(errHeaderNotJSON.Error())
		http.Error(d.w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)

		return errHeaderNotJSON
	}

	return nil
}

// decodeBody reads and decodes the request body into a TempEmailMessage struct.
// It returns an error if the body is empty or contains invalid JSON.
func (d *decoder) decodeBody(email *SMTPClient.TempEmailMessage) error {
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

// errDuringParse analyzes errors that occurred during JSON decoding,
// returns the corresponding wrapped error.
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

// checkFields checks that the fields in TempEmailMessage are not empty,
// and validates recipient email address.
func (d *decoder) checkFields(email *SMTPClient.TempEmailMessage, sendingType string) (*SMTPClient.TempEmailMessage, error) {
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

// checkTime checks the correctness of the time field and that it is in the future.
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

// convert converts data from temporary struct TempEmailMessage to EmailMessage.
func (d *decoder) convert(email *SMTPClient.TempEmailMessage) (*SMTPClient.EmailMessage, error) {
	res := &SMTPClient.EmailMessage{
		Type:    email.Type,
		To:      email.To,
		Subject: email.Subject,
		Message: email.Message,
	}

	if email.Time != "" {
		t, err := time.Parse(emailTimeLayout, email.Time)
		if err != nil {
			d.logger.Error("convert: cannot parse email.Time", zap.Error(err))
			return nil, fmt.Errorf("convert: cannot parse email.Time: %s: %w", email.Time, err)
		}

		tUnix := time.Unix(t.Unix(), 0).UTC()

		res.Time = &tUnix

	} else {
		res.Time = nil
	}

	return res, nil
}
