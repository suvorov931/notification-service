package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
)

func TestNewSendNotificationHandler(t *testing.T) {
	tests := []struct {
		name                string
		requestContext      context.Context
		body                string
		mockError           error
		wantStatusCode      int
		wantResponseMessage string
	}{
		{
			name:           "success",
			requestContext: context.Background(),
			body: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			mockError:           nil,
			wantStatusCode:      http.StatusOK,
			wantResponseMessage: "\nSuccessfully sent notification\n",
		},
		{
			name:                "error in decoder",
			requestContext:      context.Background(),
			body:                ``,
			mockError:           nil,
			wantStatusCode:      http.StatusBadRequest,
			wantResponseMessage: "Request body must not be empty\n",
		},
		{
			name:           "error in SendEmail",
			requestContext: context.Background(),
			body: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			mockError:           fmt.Errorf("SendEmail: cannot send message to"),
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
		},
		{
			name:           "context canceled during sending",
			requestContext: context.Background(),
			body: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			mockError:           context.Canceled,
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
		},
		{
			name: "context canceled before sending",
			requestContext: func() context.Context {
				canceledCtx, cancel := context.WithCancel(context.Background())
				cancel()
				return canceledCtx
			}(),
			body: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			mockError:           nil,
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/send-notification", strings.NewReader(tt.body)).WithContext(tt.requestContext)
			w := httptest.NewRecorder()
			r.Header.Set("content-type", "application/json")

			mockSender := &SMTPClient.MockEmailSender{}

			if tt.body != "" {
				mockSender.On("SendEmail", mock.Anything, mock.Anything).Return(tt.mockError)
			}

			handler := NewSendNotificationHandler(zap.NewNop(), mockSender, monitoring.NewNop())
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.Equal(t, tt.wantResponseMessage, w.Body.String())
		})
	}
}
