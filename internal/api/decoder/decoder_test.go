package decoder

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/api"
)

func TestDecoderEmailMessage(t *testing.T) {
	timeForSuccessDecodingWithTime, _ := time.ParseInLocation("2006-01-02 15:04:05", "2035-05-24 00:33:10", time.UTC)

	tests := []struct {
		name         string
		headerKey    string
		headerValue  string
		key          string
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
			key:         api.KeyForInstantSending,
			email: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &SMTPClient.EmailMessage{
				Type:    "instantSending",
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
			key:         api.KeyForInstantSending,
			email: `{
				"to": "example@gmail.com",
				"Subject": "Subject"
			}`,
			want:         nil,
			wantErr:      errNotAllFields,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Not all fields in the request body are filled in\n",
		},
		{
			name:         "empty body",
			headerKey:    "Content-Type",
			headerValue:  "application/json",
			key:          api.KeyForInstantSending,
			email:        ``,
			want:         nil,
			wantErr:      errEmptyBody,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body must not be empty\n",
		},
		{
			name:        "non json header",
			headerKey:   "Content-Type",
			headerValue: "text/plain",
			key:         api.KeyForInstantSending,
			email: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errHeaderNotJSON,
			wantStatus:   http.StatusUnsupportedMediaType,
			wantResponse: "Content-Type must be application/json\n",
		},
		{
			name:        "non content-type header",
			headerKey:   "",
			headerValue: "application/json",
			key:         api.KeyForInstantSending,
			email: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errHeaderNotJSON,
			wantStatus:   http.StatusUnsupportedMediaType,
			wantResponse: "Content-Type must be application/json\n",
		},
		{
			name:        "empty header",
			headerKey:   "",
			headerValue: "",
			key:         api.KeyForInstantSending,
			email: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errHeaderNotJSON,
			wantStatus:   http.StatusUnsupportedMediaType,
			wantResponse: "Content-Type must be application/json\n",
		},
		{
			name:        "invalid type",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForInstantSending,
			email: `{
				"to": 1.23,
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errInvalidType,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body contains an invalid value for the \"to\" field (at position 16)\n",
		},
		{
			name:        "wrong syntax",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForInstantSending,
			email: `{
				to: "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errSyntaxError,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body contains badly-formed JSON (at position 7)\n",
		},
		{
			name:        "unknown error",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForInstantSending,
			email: `{
				"to": "to",
				"subject": "subject", 
				"message": "message", 
				"": "empty field"
			}`,
			want:         nil,
			wantErr:      errUnknownError,
			wantStatus:   http.StatusInternalServerError,
			wantResponse: "Internal Server Error\n",
		},
		{
			name:        "no valid to",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForInstantSending,
			email: `{
				"to": "no-valid",
				"subject": "subject", 
				"message": "message"
			}`,
			want:         nil,
			wantErr:      errNoValidRecipientAddress,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "No valid recipient address found\n",
		},
		{
			name:        "success decoding with time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForDelayedSending,
			email: `{
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &SMTPClient.EmailMessage{
				Type:    "delayedSending",
				Time:    &timeForSuccessDecodingWithTime,
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
			key:         api.KeyForDelayedSending,
			email: `{
				"time": "2015-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errTimeNotAtFuture,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "The specified time is not in the future\n",
		},
		{
			name:        "no valid time field",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForDelayedSending,
			email: `{
				"time": "something",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errNoValidTimeField,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "The specified time is not a valid\n",
		},
		{
			name:        "invalid field time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForDelayedSending,
			email: `{
				"time": 1.23,
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      errInvalidType,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body contains an invalid value for the \"time\" field (at position 18)\n",
		},
		{
			name:        "three fields with time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForDelayedSending,
			email: `{
				"time": "2035-01-02 15:04:05",
				"to": "example@gmail.com",
				"Subject": "Subject"
			}`,
			want:         nil,
			wantErr:      errNotAllFields,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Not all fields in the request body are filled in\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.email))

			r.Header.Set(tt.headerKey, tt.headerValue)

			got, err := DecodeRequest(zap.NewNop(), r, w, tt.key)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantResponse, w.Body.String())
			assert.ErrorIs(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
