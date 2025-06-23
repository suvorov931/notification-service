package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"notification/internal/notification/service"
)

type MockRedisClient struct {
	mock.Mock
}

func (mrc *MockRedisClient) CheckRedis(ctx context.Context) ([]string, error) {
	args := mrc.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendEmail(ctx context.Context, email service.EmailMessage) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func TestWorker(t *testing.T) {
	tests := []struct {
		name           string
		redisResponse  []string
		redisError     error
		wantEmail      *service.EmailMessage
		emailError     error
		wantSendCalled bool
	}{
		{
			name:          "successful processing",
			redisResponse: []string{`{"Time":"1764687845","Email":{"to":"test@example.com","subject":"Test","message":"Test message"}}`},
			redisError:    nil,
			wantEmail: &service.EmailMessage{
				To:      "test@example.com",
				Subject: "Test",
				Message: "Test message",
			},
			emailError:     nil,
			wantSendCalled: true,
		},
		{
			name:           "empty redis response",
			redisResponse:  []string{},
			redisError:     nil,
			wantEmail:      nil,
			emailError:     nil,
			wantSendCalled: false,
		},
		{
			name:           "invalid JSON in redis response",
			redisResponse:  []string{`{invalid json}`},
			redisError:     nil,
			wantEmail:      nil,
			emailError:     nil,
			wantSendCalled: false,
		},
		{
			name:           "redis error",
			redisResponse:  nil,
			redisError:     errors.New("redis error"),
			wantEmail:      nil,
			emailError:     nil,
			wantSendCalled: false,
		},
		{
			name:          "SendEmail returns error",
			redisResponse: []string{`{"Time":"1764687845","Email":{"to":"test@example.com","subject":"Test","message":"Test message"}}`},
			redisError:    nil,
			wantEmail: &service.EmailMessage{
				To:      "test@example.com",
				Subject: "Test",
				Message: "Test message",
			},
			emailError:     errors.New("email send error"),
			wantSendCalled: true,
		},
	}

	t.Run("multiple emails", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		mockRedis := &MockRedisClient{}
		mockSender := &MockEmailSender{}

		mockRedis.On("CheckRedis", mock.Anything).Return(
			[]string{
				`{"Time":"1764687845","Email":{"to":"test1@example.com","subject":"Test1","message":"Test message1"}}`,
				`{"Time":"1764687845","Email":{"to":"test2@example.com","subject":"Test2","message":"Test message2"}}`,
			},
			nil,
		)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		wrk := New(
			zap.NewNop(),
			mockRedis,
			mockSender,
			100*time.Millisecond,
		)

		mockSender.On("SendEmail", mock.Anything, service.EmailMessage{
			To:      "test1@example.com",
			Subject: "Test1",
			Message: "Test message1",
		}).Return(nil).Run(func(args mock.Arguments) {
			wg.Done()
		})

		mockSender.On("SendEmail", mock.Anything, service.EmailMessage{
			To:      "test2@example.com",
			Subject: "Test2",
			Message: "Test message2",
		}).Return(nil).Run(func(args mock.Arguments) {
			wg.Done()
		})

		go func() {
			err := wrk.Run(ctx)
			require.NoError(t, err)
		}()

		done := make(chan struct{})
		go func() {
			wg.Wait()
			done <- struct{}{}
		}()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("SendEmail was not called for all emails in time")
		}

		cancel()
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRedis := &MockRedisClient{}
			mockSender := &MockEmailSender{}

			mockRedis.On("CheckRedis", mock.Anything).
				Return(tt.redisResponse, tt.redisError)

			wg := &sync.WaitGroup{}
			if tt.wantSendCalled {
				wg.Add(1)
				mockSender.On("SendEmail", mock.Anything, *tt.wantEmail).
					Return(tt.emailError).Run(func(args mock.Arguments) {
					wg.Done()
				})
			}

			wrk := New(
				zap.NewNop(),
				mockRedis,
				mockSender,
				100*time.Millisecond,
			)

			go func() {
				err := wrk.Run(ctx)
				require.NoError(t, err)
			}()

			if tt.wantSendCalled {
				done := make(chan struct{})

				go func() {
					wg.Wait()
					done <- struct{}{}
				}()

				select {
				case <-done:
				case <-time.After(1 * time.Second):
					t.Fatal("SendEmail was not called in time")
				}

			} else {
				time.Sleep(300 * time.Millisecond)
			}

			cancel()

			mockRedis.AssertCalled(t, "CheckRedis", mock.Anything)

			if tt.wantSendCalled && tt.wantEmail != nil {
				mockSender.AssertCalled(t, "SendEmail", mock.Anything, *tt.wantEmail)
			} else {
				mockSender.AssertNotCalled(t, "SendEmail", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestWorkerContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockRedis := &MockRedisClient{}
	mockSender := &MockEmailSender{}

	mockRedis.On("CheckRedis", mock.Anything).Return([]string{}, nil)

	wrk := New(
		zap.NewNop(),
		mockRedis,
		mockSender,
		100*time.Millisecond,
	)

	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := wrk.Run(ctx)
	require.NoError(t, err)
}
