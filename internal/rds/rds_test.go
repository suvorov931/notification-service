package rds

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"notification/internal/notification/api"
	"notification/internal/notification/service"
)

const passForRedisTestContainer = "something password"

func TestAddDelayedEmail(t *testing.T) {
	ctx := context.Background()

	addr := upRedis(ctx, "redis-for-AddDelayedEmail", t)

	cfg := &Config{
		Addr:     addr,
		Password: passForRedisTestContainer,
	}

	rc, err := New(ctx, cfg, zap.NewNop())
	require.NoError(t, err)

	tests := []struct {
		name      string
		email     *service.EmailMessageWithTime
		wantEmail []string
		wantErr   error
	}{
		{
			name: "success add",
			email: &service.EmailMessageWithTime{
				Time: "2025-12-02 15:04:05",
				Email: service.EmailMessage{
					To:      "daanisimov04@gmail.com",
					Subject: "subject",
					Message: "message",
				},
			},
			wantEmail: []string{"{\"Time\":\"1764687845\",\"Email\":{\"to\":\"daanisimov04@gmail.com\",\"subject\":\"subject\",\"message\":\"message\"}}"},
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rc.AddDelayedEmail(ctx, tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AddDelayedEmail() error = %v, wantErr %v", err, tt.wantErr)
			}

			email, err := rc.Client.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
				Min: tt.email.Time,
				Max: tt.email.Time,
			}).Result()
			require.NoError(t, err)

			assert.Equal(t, tt.wantEmail, email)
		})
	}

}

func TestCheckRedis(t *testing.T) {
	ctx := context.Background()

	addr := upRedis(ctx, "redis-for-CheckRedis", t)

	cfg := &Config{
		Addr:     addr,
		Password: passForRedisTestContainer,
	}

	rc, err := New(ctx, cfg, zap.NewNop())
	require.NoError(t, err)

	tests := []struct {
		name    string
		z       []redis.Z
		want    []string
		wantErr error
		delFunc func(rc redis.Client)
	}{
		{
			name: "success check one entry",
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "1"},
			},
			want:    []string{"1"},
			wantErr: nil,
			delFunc: func(rc redis.Client) {
				rc.ZRem(ctx, api.KeyForDelayedSending, "1")
			},
		},
		{
			name: "success check one entry",
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "2"},
				{Score: float64(time.Now().Unix()), Member: "22"},
			},
			want:    []string{"2", "22"},
			wantErr: nil,
			delFunc: func(rc redis.Client) {
				rc.ZRem(ctx, api.KeyForDelayedSending, "2", "22")
			},
		},
		{
			name: "empty entry",
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: ""},
			},
			want:    []string{""},
			wantErr: nil,
			delFunc: func(rc redis.Client) {
				rc.ZRem(ctx, api.KeyForDelayedSending)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = rc.Client.ZAdd(ctx, api.KeyForDelayedSending, tt.z...).Err()
			require.NoError(t, err)

			res, err := rc.CheckRedis(ctx)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, res)

			tt.delFunc(*rc.Client)
		})
	}
}

func TestFailedConnection(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{Addr: "localhost:1234", Password: "wrong"}

	_, err := New(ctx, cfg, zap.NewNop())
	require.Error(t, err)
}

func upRedis(ctx context.Context, containerName string, t *testing.T) string {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Name:         containerName,
		Image:        "redis:8.0",
		ExposedPorts: []string{"8021/tcp", "6379/tcp"},
		Cmd:          []string{"redis-server", "--requirepass", passForRedisTestContainer},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            false,
	})
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	return "localhost:" + port.Port()
}
