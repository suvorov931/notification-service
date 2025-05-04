package decoder

//import (
//	"io"
//	"net/http/httptest"
//	"testing"
//
//	"github.com/go-jose/go-jose/v4/json"
//
//	"notification/internal/notification/service"
//)
//
//func DecoderTest(t *testing.T) {
//	tests := []struct {
//		name    string
//		mail    []byte
//		want    service.Mail
//		wantErr error
//	}{
//		{
//			name:    "success decoding",
//			mail:    []byte(`{"to": "john@example.com"},""`),
//			want:    service.Mail{},
//			wantErr: nil,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			m, _ := io.ReadAll(tt.mail)
//			r := httptest.NewRequest("post", "/", m)
//			DecodeMailRequest()
//		})
//	}
//}
