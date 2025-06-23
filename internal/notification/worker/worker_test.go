package worker

import (
	"context"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRedis := new(MockRedisClient)
	mockSender := new(MockEmailSender)

	email := service.EmailMessage{
		To:      "test@example.com",
		Subject: "Test",
		Message: "Test message",
	}

	//emailWithTime := service.EmailMessageWithTime{
	//	Time:  "1764687845",
	//	Email: email,
	//}

	emailJSON := `{"Time":"1764687845","Email":{"to":"test@example.com","subject":"Test","message":"Test message"}}`

	mockRedis.On("CheckRedis", mock.Anything).Return([]string{emailJSON}, nil)

	sendCalled := &sync.WaitGroup{}
	sendCalled.Add(1)

	mockSender.On("SendEmail", mock.Anything, email).Return(nil).Run(func(args mock.Arguments) {
		sendCalled.Done()
	})

	wrk := Worker{
		logger:       zap.NewNop(),
		rc:           mockRedis,
		sender:       mockSender,
		tickDuration: 100 * time.Millisecond,
	}

	go func() {
		err := wrk.Run(ctx)
		require.NoError(t, err)
	}()

	sendCalled.Wait()

	cancel()

	mockRedis.AssertCalled(t, "CheckRedis", mock.Anything)
	mockSender.AssertCalled(t, "SendEmail", mock.Anything, email)
}
