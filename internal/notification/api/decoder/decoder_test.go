package decoder

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/zap"

	"notification/internal/notification/api"
	"notification/internal/notification/service"
)

func TestDecoder(t *testing.T) {
	tests := []struct {
		name         string
		headerKey    string
		headerValue  string
		key          string
		mail         string
		want         any
		wantErr      error
		wantStatus   int
		wantResponse string
	}{
		{
			name:        "success decoding",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForInstantSending,
			mail: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &service.Email{
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
			mail: `{
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
			key:          api.KeyForInstantSending,
			mail:         ``,
			want:         nil,
			wantErr:      ErrEmptyBody,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "Request body must not be empty\n",
		},
		{
			name:        "non json header",
			headerKey:   "Content-Type",
			headerValue: "text/plain",
			key:         api.KeyForInstantSending,
			mail: `{
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
			key:         api.KeyForInstantSending,
			mail: `{
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
			key:         api.KeyForInstantSending,
			mail: `{
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
			key:         api.KeyForInstantSending,
			mail: `{
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
			key:         api.KeyForInstantSending,
			mail: `{
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
			key:         api.KeyForInstantSending,
			mail: `{
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
			key:         api.KeyForInstantSending,
			mail: `{
				"to": "no-valid",
				"subject": "subject", 
				"message": "message"
			}`,
			want:         nil,
			wantErr:      ErrNoValidRecipientAddress,
			wantStatus:   http.StatusBadRequest,
			wantResponse: "No valid recipient address found\n",
		},
		{
			name:        "invalid key",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         "invalidKey",
			mail: `{
				"time": "2025-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:         nil,
			wantErr:      ErrUnknownKey,
			wantStatus:   http.StatusInternalServerError,
			wantResponse: "Internal Server Error\n",
		},
		{
			name:        "success decoding with time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         api.KeyForDelayedSending,
			mail: `{
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &service.EmailWithTime{
				Time: "2035-05-24 00:33:10",
				Email: service.Email{
					To:      "example@gmail.com",
					Subject: "Subject",
					Message: "Message",
				},
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
			mail: `{
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
			key:         api.KeyForDelayedSending,
			mail: `{
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
			key:         api.KeyForDelayedSending,
			mail: `{
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
			key:         api.KeyForDelayedSending,
			mail: `{
				"To": "To",
				"Subject": "Subject",
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
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.mail))

			r.Header.Set(tt.headerKey, tt.headerValue)

			gotAny, err := DecodeEmailRequest(tt.key, w, r, zap.NewNop())

			if w.Code != tt.wantStatus {
				t.Errorf("DecodeMailRequest(): status = %d, wantStatus = %d", w.Code, tt.wantStatus)
			}

			if w.Body.String() != tt.wantResponse {
				t.Errorf("DecodeMailRequest(): response = %s, wantResponse = %s", w.Body.String(), tt.wantResponse)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DecodeMailRequest(): error = %v, wantErr = %v", err, tt.wantErr)
			}

			if err == nil {
				switch tt.key {
				case api.KeyForInstantSending:
					got, ok := gotAny.(*service.Email)
					if !ok {
						t.Errorf("DecodeMailRequest(): expected *service.Email, got %T", gotAny)
						return
					}

					want, ok := tt.want.(*service.Email)
					if !ok {
						t.Errorf("Test setup error: want is not *service.Email, got %T", tt.want)
						return
					}

					if !reflect.DeepEqual(want, got) {
						t.Errorf("DecodeMailRequest(): got = %v, want = %v", got, tt.want)
					}

				case api.KeyForDelayedSending:
					got, ok := gotAny.(*service.EmailWithTime)
					if !ok {
						t.Errorf("DecodeMailRequest(): expected *service.EmailWithTime, got %T", gotAny)
						return
					}

					want, ok := tt.want.(*service.EmailWithTime)
					if !ok {
						t.Errorf("Test setup error: want is not *service.EmailWithTime, got %T", tt.want)
						return
					}

					if !reflect.DeepEqual(got, want) {
						t.Errorf("DecodeMailRequest(): got = %v, want = %v", got, tt.want)
					}
				}
			}
		})
	}
}
