package handlers

//
//import (
//	"context"
//	"fmt"
//	"net/http"
//	"net/http/httptest"
//	"strings"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/mock"
//	"go.uber.org/zap"
//
//	"notification/internal/SMTPClient"
//	"notification/internal/monitoring"
//	"notification/internal/storage/redisClient"
//)
//
//func TestNewSendNotificationHandler(t *testing.T) {
//	tests := []struct {
//		name                string
//		requestContext      context.Context
//		body                string
//		mockError           error
//		wantStatusCode      int
//		wantResponseMessage string
//	}{
//		{
//			name:           "success",
//			requestContext: context.Background(),
//			body: `{
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			mockError:           nil,
//			wantStatusCode:      http.StatusOK,
//			wantResponseMessage: "\nSuccessfully sent notification\n",
//		},
//		{
//			name:                "error in decoder",
//			requestContext:      context.Background(),
//			body:                ``,
//			mockError:           nil,
//			wantStatusCode:      http.StatusBadRequest,
//			wantResponseMessage: "Request body must not be empty\n",
//		},
//		{
//			name:           "error in SendEmail",
//			requestContext: context.Background(),
//			body: `{
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			mockError:           fmt.Errorf("SendEmail: cannot send message to"),
//			wantStatusCode:      http.StatusInternalServerError,
//			wantResponseMessage: http.StatusText(500) + "\n",
//		},
//		{
//			name:           "context canceled during sending",
//			requestContext: context.Background(),
//			body: `{
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			mockError:           context.Canceled,
//			wantStatusCode:      http.StatusInternalServerError,
//			wantResponseMessage: http.StatusText(500) + "\n",
//		},
//		{
//			name: "context canceled before sending",
//			requestContext: func() context.Context {
//				canceledCtx, cancel := context.WithCancel(context.Background())
//				cancel()
//				return canceledCtx
//			}(),
//			body: `{
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			mockError:           nil,
//			wantStatusCode:      http.StatusInternalServerError,
//			wantResponseMessage: http.StatusText(500) + "\n",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			r := httptest.NewRequest("POST", "/send-notification", strings.NewReader(tt.body)).WithContext(tt.requestContext)
//			w := httptest.NewRecorder()
//			r.Header.Set("content-type", "application/json")
//
//			mockSender := &SMTPClient.MockEmailSender{}
//
//			if tt.body != "" {
//				mockSender.On("SendEmail", mock.Anything, mock.Anything).Return(tt.mockError)
//			}
//
//			handler := NewSendNotificationHandler(mockSender, zap.NewNop(), monitoring.NewNop())
//			handler.ServeHTTP(w, r)
//
//			assert.Equal(t, tt.wantStatusCode, w.Code)
//			assert.Equal(t, tt.wantResponseMessage, w.Body.String())
//		})
//	}
//}
//
//func TestNewSendNotificationViaTimeHandler(t *testing.T) {
//	tests := []struct {
//		name                string
//		requestContext      context.Context
//		body                string
//		email               *SMTPClient.EmailMessage
//		mockError           error
//		wantStatusCode      int
//		wantResponseMessage string
//	}{
//		{
//			name:           "success",
//			requestContext: context.Background(),
//			body: `{
//				"time":"2035-07-01 16:36:00",
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			email: &SMTPClient.EmailMessage{
//				To:      "example@gmail.com",
//				Subject: "Subject",
//				Message: "Message",
//			},
//			mockError:           nil,
//			wantStatusCode:      http.StatusOK,
//			wantResponseMessage: "\nSuccessfully saved your mail\n",
//		},
//		{
//			name:                "error in decoder",
//			requestContext:      context.Background(),
//			body:                ``,
//			mockError:           nil,
//			wantStatusCode:      http.StatusBadRequest,
//			wantResponseMessage: "Request body must not be empty\n",
//		},
//		{
//			name:           "error in AddDelayedEmail",
//			requestContext: context.Background(),
//			body: `{
//				"time":"2035-07-01 16:36:00",
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			mockError:           fmt.Errorf("SendEmail: cannot send message to"),
//			wantStatusCode:      http.StatusInternalServerError,
//			wantResponseMessage: http.StatusText(500) + "\n",
//		},
//		{
//			name:           "context canceled during add",
//			requestContext: context.Background(),
//			body: `{
//				"time":"2035-07-01 16:36:00",
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			mockError:           context.Canceled,
//			wantStatusCode:      http.StatusInternalServerError,
//			wantResponseMessage: http.StatusText(500) + "\n",
//		},
//		{
//			name: "context canceled before add",
//			requestContext: func() context.Context {
//				canceledCtx, cancel := context.WithCancel(context.Background())
//				cancel()
//				return canceledCtx
//			}(),
//			body: `{
//				"time":"2035-07-01 16:36:00",
//				"to": "example@gmail.com",
//				"subject": "Subject",
//				"message": "Message"
//			}`,
//			mockError:           nil,
//			wantStatusCode:      http.StatusInternalServerError,
//			wantResponseMessage: http.StatusText(500) + "\n",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			r := httptest.NewRequest("POST", "/send-notification", strings.NewReader(tt.body)).WithContext(tt.requestContext)
//			w := httptest.NewRecorder()
//			r.Header.Set("content-type", "application/json")
//
//			mockRedis := &redisClient.MockRedisClient{}
//
//			if tt.mockError != nil || tt.wantStatusCode == http.StatusOK {
//				mockRedis.On("AddDelayedEmail", mock.Anything, mock.Anything).Return(tt.mockError)
//			}
//
//			handler := NewSendNotificationViaTimeHandler(mockRedis, zap.NewNop(), monitoring.NewNop())
//			handler.ServeHTTP(w, r)
//
//			assert.Equal(t, tt.wantStatusCode, w.Code)
//			assert.Equal(t, tt.wantResponseMessage, w.Body.String())
//		})
//	}
//}
