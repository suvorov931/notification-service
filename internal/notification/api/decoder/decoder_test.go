package decoder

import (
	"testing"

	"notification/internal/notification/service"
)

func DecoderTest(t *testing.T) {
	tests := []struct {
		name    string
		mail    service.Mail
		want    service.Mail
		wantErr error
	}{
		{
			name: "success decoding",
			mail: service.Mail{
				To:      "To",
				Subject: "Subject",
				Message: "Message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DecodeMailRequest()
		})
	}
}
