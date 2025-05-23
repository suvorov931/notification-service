package decoder

import (
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/zap"

	"notification/internal/notification/service"
)

func TestDecoder(t *testing.T) {
	tests := []struct {
		name        string
		headerKey   string
		headerValue string
		key         string
		mail        string
		want        any
		wantErr     error
	}{
		{
			name:        "success decoding",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForInstantSending,
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
			wantErr: nil,
		},
		{
			name:        "two fields",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForInstantSending,
			mail: `{
				"to": "example@gmail.com",
				"Subject": "Subject"
			}`,
			want:    nil,
			wantErr: ErrNotAllFields,
		},
		{
			name:        "empty body",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForInstantSending,
			mail:        ``,
			want:        nil,
			wantErr:     ErrEmptyBody,
		},
		{
			name:        "non json header",
			headerKey:   "Content-Type",
			headerValue: "text/plain",
			key:         KeyForInstantSending,
			mail: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:    nil,
			wantErr: ErrHeaderNotJSON,
		},
		{
			name:        "non content-type header",
			headerKey:   "",
			headerValue: "application/json",
			key:         KeyForInstantSending,
			mail: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:    nil,
			wantErr: ErrHeaderNotJSON,
		},
		{
			name:        "empty header",
			headerKey:   "",
			headerValue: "",
			key:         KeyForInstantSending,
			mail: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:    nil,
			wantErr: ErrHeaderNotJSON,
		},
		{
			name:        "invalid type",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForInstantSending,
			mail: `{
				"to": 1.23,
				"subject": "Subject",
				"message": "Message"
			}`,
			want:    nil,
			wantErr: ErrInvalidType,
		},
		{
			name:        "wrong syntax",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForInstantSending,
			mail: `{
				to: "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:    nil,
			wantErr: ErrSyntaxError,
		},
		{
			name:        "unknown error",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForInstantSending,
			mail: `{
				"to": "to",
				"subject": "subject", 
				"message": "message", 
				"": "empty field"
			}`,
			want:    nil,
			wantErr: ErrUnknownError,
		},
		{
			name:        "no valid to",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForInstantSending,
			mail: `{
				"to": "no-valid",
				"subject": "subject", 
				"message": "message"
			}`,
			want:    nil,
			wantErr: ErrNoValidRecipientAddress,
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
			want:    nil,
			wantErr: ErrUnknownKey,
		},
		{
			name:        "success decoding with time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForDelayedSending,
			mail: `{
				"time": "2025-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &service.EmailWithTime{
				Time:    "2025-05-24 00:33:10",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			wantErr: nil,
		},
		{
			name:        "two fields with time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForDelayedSending,
			mail: `{
				"To": "To",
				"Subject": "Subject"
			}`,
			want:    nil,
			wantErr: ErrNotAllFields,
		},
		{
			name:        "invalid field time",
			headerKey:   "Content-Type",
			headerValue: "application/json",
			key:         KeyForDelayedSending,
			mail: `{
				"time": 1.23,
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:    nil,
			wantErr: ErrInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.mail))

			r.Header.Set(tt.headerKey, tt.headerValue)

			gotAny, err := DecodeEmailRequest(tt.key, w, r, zap.NewNop())

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DecodeMailRequest(): error = %v, wantErr = %v", err, tt.wantErr)
			}

			if err == nil {
				switch tt.key {
				case KeyForInstantSending:
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

				case KeyForDelayedSending:
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

					if !reflect.DeepEqual(want, got) {
						t.Errorf("DecodeMailRequest(): got = %v, want = %v", got, tt.want)
					}
				}
			}
		})
	}
}
