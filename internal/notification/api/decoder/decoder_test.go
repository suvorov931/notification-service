package decoder

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/zap"

	"notification/internal/notification/service"
)

func TestDecoder(t *testing.T) {
	tests := []struct {
		name    string
		mail    string
		want    *service.Mail
		wantErr error
	}{
		{
			name: "success decoding",
			//header: "application/json",
			mail: `{
				"to": "To",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &service.Mail{
				To:      "To",
				Subject: "Subject",
				Message: "Message",
			},
			wantErr: nil,
		},
		{
			name: "two fields",
			mail: `{
				"Subject": "Subject",
				"message": "Message"
			}`,
			want:    nil,
			wantErr: ErrNotAllFields,
		},
		{
			name:    "empty body",
			mail:    ``,
			want:    nil,
			wantErr: ErrEmptyBody,
		},
		{
			name: "two fields",
			mail: "to: To" +
				"subject: Subject" +
				"message: Message",
			want:    nil,
			wantErr: ErrNotAllFields,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.mail))
			r.Header.Set("Content-Type", "application/json")

			got, err := DecodeMailRequest(w, r, zap.NewNop())
			fmt.Println(got)
			fmt.Println(err)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DecodeMailRequest(): error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("DecodeMailRequest(): got = %v, want %v", got, tt.want)
			}
		})
	}
}
