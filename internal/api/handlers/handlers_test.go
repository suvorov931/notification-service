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
	"notification/internal/api"
	"notification/internal/monitoring"
	"notification/internal/storage/postgresClient"
	"notification/internal/storage/redisClient"
)

func TestNewSendNotificationHandler(t *testing.T) {
	tests := []struct {
		name                string
		requestContext      context.Context
		body                string
		email               SMTPClient.EmailMessage
		id                  int
		postgresError       error
		senderError         error
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
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForInstantSending,
				Time:    "",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			id:                  1,
			postgresError:       nil,
			senderError:         nil,
			wantStatusCode:      http.StatusOK,
			wantResponseMessage: "{\"message\":\"Successfully sent notification\",\"id\":1}\n",
		},
		{
			name:                "error in decoder",
			requestContext:      context.Background(),
			body:                ``,
			senderError:         nil,
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
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForInstantSending,
				Time:    "",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			senderError:         fmt.Errorf("SendEmail: cannot send message to"),
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
		},
		{
			name:           "error in SavingInstantSending",
			requestContext: context.Background(),
			body: `{
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForInstantSending,
				Time:    "",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			id:                  0,
			postgresError:       fmt.Errorf("SavingInstantSending: failed to add email to database"),
			senderError:         nil,
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
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForInstantSending,
				Time:    "",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			senderError:         context.Canceled,
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
			senderError:         nil,
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
			mockRedisClient := &redisClient.MockRedisClient{}
			mockPostgresClient := &postgresClient.MockPostgresService{}

			notificationHandler := New(
				zap.NewNop(),
				mockSender,
				mockRedisClient,
				mockPostgresClient,
			)

			if tt.body != "" {
				mockSender.On("SendEmail", mock.Anything, tt.email).Return(tt.senderError)
				mockPostgresClient.On("SaveEmail", mock.Anything, &tt.email).Return(tt.id, tt.postgresError)

			}

			handler := notificationHandler.NewSendNotificationHandler(monitoring.NewNop())
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.Equal(t, tt.wantResponseMessage, w.Body.String())
		})
	}
}

func TestNewSendNotificationViaTimeHandler(t *testing.T) {
	tests := []struct {
		name                string
		requestContext      context.Context
		body                string
		email               SMTPClient.EmailMessage
		id                  int
		postgresError       error
		redisError          error
		wantStatusCode      int
		wantResponseMessage string
	}{
		{
			name:           "success",
			requestContext: context.Background(),
			body: `{
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    "2035-05-24 00:33:10",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			id:                  1,
			postgresError:       nil,
			redisError:          nil,
			wantStatusCode:      http.StatusOK,
			wantResponseMessage: "{\"message\":\"Successfully saved your mail\",\"id\":1}\n",
		},
		{
			name:                "error in decoder",
			requestContext:      context.Background(),
			body:                ``,
			redisError:          nil,
			wantStatusCode:      http.StatusBadRequest,
			wantResponseMessage: "Request body must not be empty\n",
		},
		{
			name:           "error in SendEmail",
			requestContext: context.Background(),
			body: `{
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    "2035-05-24 00:33:10",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			redisError:          fmt.Errorf("SendEmail: cannot send message to"),
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
		},
		{
			name:           "error in SavingInstantSending",
			requestContext: context.Background(),
			body: `{
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    "2035-05-24 00:33:10",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			id:                  0,
			postgresError:       fmt.Errorf("SavingInstantSending: failed to add email to database"),
			redisError:          nil,
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
		},
		{
			name:           "context canceled during sending",
			requestContext: context.Background(),
			body: `{
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			email: SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    "2035-05-24 00:33:10",
				To:      "example@gmail.com",
				Subject: "Subject",
				Message: "Message",
			},
			redisError:          context.Canceled,
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
				"time": "2035-05-24 00:33:10",
				"to": "example@gmail.com",
				"subject": "Subject",
				"message": "Message"
			}`,
			redisError:          nil,
			wantStatusCode:      http.StatusInternalServerError,
			wantResponseMessage: http.StatusText(500) + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/send-notification-via-time", strings.NewReader(tt.body)).WithContext(tt.requestContext)
			w := httptest.NewRecorder()
			r.Header.Set("content-type", "application/json")

			mockSender := &SMTPClient.MockEmailSender{}
			mockRedisClient := &redisClient.MockRedisClient{}
			mockPostgresClient := &postgresClient.MockPostgresService{}

			notificationHandler := New(
				zap.NewNop(),
				mockSender,
				mockRedisClient,
				mockPostgresClient,
			)

			if tt.body != "" {
				mockRedisClient.On("AddDelayedEmail", mock.Anything, &tt.email).Return(tt.redisError)
				mockPostgresClient.On("SaveEmail", mock.Anything, &tt.email).Return(tt.id, tt.postgresError)
			}

			handler := notificationHandler.NewSendNotificationViaTimeHandler(monitoring.NewNop())
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.Equal(t, tt.wantResponseMessage, w.Body.String())
		})
	}
}

func TestNewListNotificationHandler(t *testing.T) {
	tests := []struct {
		name                string
		requestContext      context.Context
		email               []*SMTPClient.EmailMessage
		query               string
		id                  int
		wantError           error
		wantStatusCode      int
		wantResponseMessage string
	}{
		{
			name:           "success for instantSending",
			requestContext: context.Background(),
			email: []*SMTPClient.EmailMessage{&SMTPClient.EmailMessage{
				Type:    "instantSending",
				Time:    "",
				To:      "to",
				Subject: "subject",
				Message: "message",
			}},
			query:               "/list?by=id&id=1",
			id:                  1,
			wantError:           nil,
			wantStatusCode:      http.StatusOK,
			wantResponseMessage: "[{\"type\":\"instantSending\",\"to\":\"to\",\"subject\":\"subject\",\"message\":\"message\"}]\n",
		},
		{
			name:           "success for delayedSending",
			requestContext: context.Background(),
			email: []*SMTPClient.EmailMessage{&SMTPClient.EmailMessage{
				Type:    "delayedSending",
				Time:    "time",
				To:      "to",
				Subject: "subject",
				Message: "message",
			}},
			query:               "/list?by=id&id=2",
			id:                  2,
			wantError:           nil,
			wantStatusCode:      http.StatusOK,
			wantResponseMessage: "[{\"type\":\"delayedSending\",\"time\":\"time\",\"to\":\"to\",\"subject\":\"subject\",\"message\":\"message\"}]\n",
		},
		{
			name:                "invalid query",
			requestContext:      context.Background(),
			query:               "/list?invalidQuery",
			wantStatusCode:      http.StatusBadRequest,
			wantResponseMessage: ErrInvalidQuery.Error() + "\n",
		},
	}
	// TODO: валидация query
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.query, nil).WithContext(tt.requestContext)
			w := httptest.NewRecorder()
			r.Header.Set("content-type", "application/json")

			mockSender := &SMTPClient.MockEmailSender{}
			mockRedisClient := &redisClient.MockRedisClient{}
			mockPostgresClient := &postgresClient.MockPostgresService{}

			notificationHandler := New(
				zap.NewNop(),
				mockSender,
				mockRedisClient,
				mockPostgresClient,
			)

			mockPostgresClient.On("FetchById", mock.Anything, tt.id).Return(tt.email, tt.wantError)

			handler := notificationHandler.NewListNotificationHandler(monitoring.NewNop())
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.Equal(t, tt.wantResponseMessage, w.Body.String())
		})
	}
}
