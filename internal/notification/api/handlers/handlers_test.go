package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/monitoring"
	"notification/internal/notification/SMTPClient"
	"notification/internal/notification/api/decoder"
)

func TestNewSendNotificationHandler(t *testing.T) {
	tests := []struct {
		name                string
		body                string
		want                *SMTPClient.EmailMessage
		wantStatusCode      int
		wantResponseMessage string
		wantError           error
	}{
		{
			name: "success",
			body: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want: &SMTPClient.EmailMessage{
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			wantStatusCode:      http.StatusOK,
			wantResponseMessage: "Successfully sent notification\n",
			wantError:           nil,
		},
		{
			name:                "error in decoder",
			body:                ``,
			want:                nil,
			wantStatusCode:      http.StatusBadRequest,
			wantResponseMessage: "Request body must not be empty\n",
			wantError:           decoder.ErrEmptyBody,
		},
		{
			name: "error in SendEmail",
			body: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			want:                nil,
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
			wantError:           fmt.Errorf("SendEmail: cannot send message to"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/send-notification", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			r.Header.Set("content-type", "application/json")

			mockSender := &SMTPClient.MockEmailSender{}

			// if tt.want != nil {
			// mockSender.On("SendEmail", mock.Anything, *tt.want).Return(tt.wantError)
			// }
			mockSender.On("SendEmail", mock.Anything, mock.Anything).Return(tt.wantError)

			handler := NewSendNotificationHandler(zap.NewNop(), mockSender, monitoring.NewNop())

			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.Equal(t, tt.wantResponseMessage, w.Body.String())
			// mockSender.AssertNotCalled(t, "SendEmail")
		})
	}
}
