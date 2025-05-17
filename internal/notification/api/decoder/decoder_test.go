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
		mail        string
		want        *service.Email
		wantErr     error
	}{
		{
			name:        "success decoding",
			headerKey:   "Content-Type",
			headerValue: "application/json",
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
			mail:        ``,
			want:        nil,
			wantErr:     ErrEmptyBody,
		},
		{
			name:        "non json header",
			headerKey:   "Content-Type",
			headerValue: "text/plain",
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
			mail: `{
				"to": "no-valid",
				"subject": "subject", 
				"message": "message"
			}`,
			want:    nil,
			wantErr: ErrNoValidRecipientAddress,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.mail))

			r.Header.Set(tt.headerKey, tt.headerValue)

			got, err := DecodeMailRequest(w, r, zap.NewNop())

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DecodeMailRequest(): error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("DecodeMailRequest(): got = %v, want %v", got, tt.want)
			}
		})
	}
}
