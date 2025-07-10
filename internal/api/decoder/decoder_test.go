package decoder

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
)

func TestDecoderEmailMessage(t *testing.T) {
	tests := []struct {
		name         string
		headerKey    string
		headerValue  string
		email        string
		want         *SMTPClient.EmailMessage
		wantErr      error
		wantStatus   int
		wantResponse string
	}{
		{
			name:        "success decoding",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &SMTPClient.EmailMessage{
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			wantErr:      nil,
			wantStatus:   http.StatusOK,
			wantResponse: "",
		},
		{
			name:        "two fields",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"to": "example@gmail.com",
				"Subject": "Subject"
			}`,
			want:         nil,
			wantErr:      ErrNotAllFields,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Not all fields in the request body are filled in\n",
		},
		{
			name:         "empty body",
			headerKey:    "Content-Type",
			headerValue:  "application/json",
			email:        ``,
			want:         nil,
			wantErr:      ErrEmptyBody,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body must not be empty\n",
		},
		{
			name:        "non json header",
			headerKey:   "Content-Type",
			headerValue: "text/plain",
			email: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrHeaderNotJSON,
			wantStatus:   http.StatusUnsupportedMediaType,
			wantResponse: "Content-Type must be application/json\n",
		},
		{
			name:        "non content-type header",
			headerKey:   "",
			headerValue: "application/json",
			email: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrHeaderNotJSON,
			wantStatus:   http.StatusUnsupportedMediaType,
			wantResponse: "Content-Type must be application/json\n",
		},
		{
			name:        "empty header",
			headerKey:   "",
			headerValue: "",
			email: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrHeaderNotJSON,
			wantStatus:   http.StatusUnsupportedMediaType,
			wantResponse: "Content-Type must be application/json\n",
		},
		{
			name:        "invalid type",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"to": 1.23,
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrInvalidType,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body contains an invalid value for the \"to\" field (at position 16)\n",
		},
		{
			name:        "wrong syntax",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				to: "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrSyntaxError,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body contains badly-formed JSON (at position 7)\n",
		},
		{
			name:        "unknown error",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"to": "to",
				"subject": "subject", 
				"message": "message", 
				"": "empty field"
			}`,
			want:         nil,
			wantErr:      ErrUnknownError,
			wantStatus:   http.StatusInternalServerError,
			wantResponse: "Internal Server Error\n",
		},
		{
			name:        "no valid to",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"to": "no-valid",
				"subject": "subject", 
				"message": "message"
			}`,
			want:         nil,
			wantErr:      ErrNoValidRecipientAddress,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "No valid recipient address found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.email))

			r.Header.Set(tt.headerKey, tt.headerValue)

			got, err := Decoder[SMTPClient.EmailMessage]{zap.NewNop(), r, w}.Decode()

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantResponse, w.Body.String())
			assert.ErrorIs(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecoderEmailMessageWithTime(t *testing.T) {
	tests := []struct {
		name         string
		headerKey    string
		headerValue  string
		email        string
		want         *SMTPClient.EmailMessageWithTime
		wantErr      error
		wantStatus   int
		wantResponse string
	}{
		{
			name:        "success decoding with time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &SMTPClient.EmailMessageWithTime{
				Time:    "2035-05-24 00:33:10",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			wantErr:      nil,
			wantStatus:   http.StatusOK,
			wantResponse: "",
		},
		{
			name:        "time not at future",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"time": "2015-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrTimeNotAtFuture,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "The specified time is not in the future\n",
		},
		{
			name:        "no valid time field",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"time": "something",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrNoValidTimeFiled,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "The specified time is not a valid\n",
		},
		{
			name:        "invalid field time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"time": 1.23,
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrInvalidType,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body contains an invalid value for the \"time\" field (at position 18)\n",
		},
		{
			name:        "three fields with time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			email: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrNotAllFields,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Not all fields in the request body are filled in\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.email))

			r.Header.Set(tt.headerKey, tt.headerValue)

			got, err := Decoder[SMTPClient.EmailMessageWithTime]{zap.NewNop(), r, w}.Decode()

			assert.Equal(t, w.Code, tt.wantStatus)
			assert.Equal(t, w.Body.String(), tt.wantResponse)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
