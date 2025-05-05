package decoder

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"notification/internal/notification/service"
)

func TestDecoder(t *testing.T) {
	tests := []struct {
		name    string
		mail    string
		want    service.Mail
		wantErr error
	}{
		{
			name: "success decoding",
			mail: `{
				"to": "To", 
				"subject": "Subject", 
				"message": "Message"
			}`,
			want: service.Mail{
				To:      "To",
				Subject: "Subject",
				Message: "Message",
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.mail))
			w := &mockResponseWriter{headers: make(http.Header)}
			got, err := DecodeMailRequest(w, r, zap.NewNop())

			if errors.Is(err, tt.wantErr) {
				t.Errorf("DecodeMailRequest(): error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DecodeMailRequest(): got = %v, want %v", got, tt.want)
			}
		})
	}
}
