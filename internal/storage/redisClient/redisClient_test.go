package redisClient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/api"
	"notification/internal/monitoring"
)

func TestAddDelayedEmail(t *testing.T) {
	timeForSuccessAdd, _ := time.ParseInLocation("2006-01-02 15:04:05", "2025-12-02 15:04:05", time.UTC)

	addrs := upRedisCluster(context.Background(), "TestAddDelayedEmail", 1, t)

	rc, err := New(context.Background(), &Config{Addrs: addrs}, monitoring.NewNop(), zap.NewNop())
	require.NoError(t, err)

	tests := []struct {
		name      string
		ctx       context.Context
		email     *SMTPClient.EmailMessage
		wantEmail []string
		wantErr   error
	}{
		{
			name: "success add",
			ctx:  context.Background(),
			email: &SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    &timeForSuccessAdd,
				To:      "daanisimov04@gmail.com",
				Subject: "subject",
				Message: "message",
			},
			wantEmail: []string{"{\"type\":\"delayedSending\",\"time\":\"1764687845\",\"to\":\"daanisimov04@gmail.com\",\"subject\":\"subject\",\"message\":\"message\"}"},
			wantErr:   nil,
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			email: &SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    &timeForSuccessAdd,
				To:      "daanisimov04@gmail.com",
				Subject: "subject",
				Message: "message",
			},
			wantEmail: nil,
			wantErr:   context.Canceled,
		},
		{
			name: "context deadline exceeded",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(2 * time.Nanosecond)
				return ctx
			}(),
			email: &SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    &timeForSuccessAdd,
				To:      "daanisimov04@gmail.com",
				Subject: "subject",
				Message: "message",
			},
			wantEmail: nil,
			wantErr:   context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = rc.AddDelayedEmail(tt.ctx, tt.email)
			require.ErrorIs(t, err, tt.wantErr)

			email, err := rc.cluster.ZRangeByScore(tt.ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
				Min: strconv.FormatInt(tt.email.Time.Unix(), 10),
				Max: strconv.FormatInt(tt.email.Time.Unix(), 10),
			}).Result()
			require.ErrorIs(t, err, tt.wantErr)

			assert.Equal(t, tt.wantEmail, email)
		})
	}

}

func TestCheckRedis(t *testing.T) {
	addrs := upRedisCluster(context.Background(), "TestCheckRedis", 2, t)

	rc, err := New(context.Background(), &Config{Addrs: addrs}, monitoring.NewNop(), zap.NewNop())
	require.NoError(t, err)

	tests := []struct {
		name    string
		ctx     context.Context
		z       []redis.Z
		want    []string
		wantErr error
		delFunc func(ctx context.Context, rc redis.ClusterClient)
	}{
		{
			name: "success check one entry",
			ctx:  context.Background(),
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "1"},
			},
			want:    []string{"1"},
			wantErr: nil,
			delFunc: func(ctx context.Context, rc redis.ClusterClient) {
				rc.ZRem(ctx, api.KeyForDelayedSending, "1")
			},
		},
		{
			name: "success check two entry",
			ctx:  context.Background(),
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "2"},
				{Score: float64(time.Now().Unix()), Member: "22"},
			},
			want:    []string{"2", "22"},
			wantErr: nil,
			delFunc: func(ctx context.Context, rc redis.ClusterClient) {
				rc.ZRem(ctx, api.KeyForDelayedSending, "2", "22")
			},
		},
		{
			name: "empty entry",
			ctx:  context.Background(),
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: ""},
			},
			want:    []string{""},
			wantErr: nil,
			delFunc: func(ctx context.Context, rc redis.ClusterClient) {
				rc.ZRem(ctx, api.KeyForDelayedSending)
			},
		},
		{
			name: "context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "1"},
			},
			want:    []string(nil),
			wantErr: context.Canceled,
			delFunc: func(ctx context.Context, rc redis.ClusterClient) {
				rc.ZRem(ctx, api.KeyForDelayedSending, "1")
			},
		},
		{
			name: "context deadline exceeded",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(2 * time.Nanosecond)
				return ctx
			}(),
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "1"},
			},
			want:    []string(nil),
			wantErr: context.DeadlineExceeded,
			delFunc: func(ctx context.Context, rc redis.ClusterClient) {
				rc.ZRem(ctx, api.KeyForDelayedSending, "1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = rc.cluster.ZAdd(context.Background(), api.KeyForDelayedSending, tt.z...).Err()
			require.NoError(t, err)

			res, err := rc.CheckRedis(tt.ctx)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, res)

			tt.delFunc(context.Background(), *rc.cluster)
		})
	}

	t.Run("check removal after reading", func(t *testing.T) {
		ctx := context.Background()

		err = rc.cluster.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: "something",
		}).Err()
		require.NoError(t, err)

		res, err := rc.CheckRedis(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"something"}, res)

		emptyRes, err := rc.cluster.ZRange(ctx, api.KeyForDelayedSending, 0, -1).Result()
		require.NoError(t, err)
		assert.Equal(t, []string{}, emptyRes)

	})
}

func TestFailedConnection(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		Addrs: []string{
			"localhost:1234",
		},
		Password: "wrong",
	}

	_, err := New(ctx, cfg, monitoring.NewNop(), zap.NewNop())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}

func TestClose(t *testing.T) {
	tests := []struct {
		name            string
		shutdownTimeout time.Duration
		wantErr         error
	}{
		{
			name:            "success",
			shutdownTimeout: 3 * time.Second,
			wantErr:         nil,
		},
		{
			name:            "success",
			shutdownTimeout: 0 * time.Second,
			wantErr:         context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := redismock.NewClusterMock()
			rc := RedisCluster{
				cluster:         db,
				metrics:         monitoring.NewNop(),
				logger:          zap.NewNop(),
				shutdownTimeout: tt.shutdownTimeout,
			}

			err := rc.Close()
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestParseAndConvertTime(t *testing.T) {
	rc := RedisCluster{
		logger: zap.NewNop(),
	}

	testTime := time.Date(2035, 6, 27, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		email   *SMTPClient.EmailMessage
		wantErr error
	}{
		{
			name: "success",
			email: &SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    &testTime,
				To:      "test@gmail.com",
				Subject: "subject",
				Message: "message",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, score, err := rc.parseAndConvertTime(tt.email)
			assert.ErrorIs(t, err, tt.wantErr)

			wantScore := float64(testTime.Unix())

			var resultStruct SMTPClient.TempEmailMessage
			err = json.Unmarshal(res, &resultStruct)
			require.NoError(t, err)

			assert.Equal(t, wantScore, score)
			assert.Equal(t, tt.email.Type, resultStruct.Type)
			assert.Equal(t, strconv.FormatInt(testTime.Unix(), 10), resultStruct.Time)
			assert.Equal(t, tt.email.To, resultStruct.To)
			assert.Equal(t, tt.email.Subject, resultStruct.Subject)
			assert.Equal(t, tt.email.Message, resultStruct.Message)

		})
	}
}

func upRedisCluster(ctx context.Context, containerName string, num int, t *testing.T) []string {
	t.Helper()

	addrs := make([]string, 0, 6)

	for i := 1; i <= 6; i++ {
		port := fmt.Sprintf("70%d%d", num, i)
		busPort := fmt.Sprintf("170%d%d", num, i)

		req := testcontainers.ContainerRequest{
			Name:  fmt.Sprintf("%s-%d", containerName, i),
			Image: "redis:8.0",
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.NetworkMode = "host"
			},
			Cmd: []string{
				"redis-server",
				"--port", port,
				"--cluster-announce-ip", "127.0.0.1",
				"--cluster-announce-port", port,
				"--cluster-announce-bus-port", busPort,
				"--cluster-enabled", "yes",
				"--cluster-config-file", "nodes.conf",
				"--cluster-node-timeout", "5000",
				"--appendonly", "yes",
			},
			WaitingFor: wait.ForLog("Ready to accept connections"),
		}

		cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
			Reuse:            false,
		})
		require.NoError(t, err)

		addr := fmt.Sprintf("127.0.0.1:%s", port)
		addrs = append(addrs, addr)

		if len(addrs) == 6 {
			cmd := append([]string{"redis-cli", "--cluster", "create"}, addrs...)
			cmd = append(cmd, "--cluster-replicas", "1", "--cluster-yes")

			_, _, err := cont.Exec(ctx, cmd)
			require.NoError(t, err)

			time.Sleep(time.Second)

			_, r, err := cont.Exec(ctx, []string{
				"redis-cli", "-p", port, "cluster", "info",
			})
			require.NoError(t, err)

			b, err := io.ReadAll(r)
			require.NoError(t, err)

			if !strings.Contains(string(b), "cluster_state:ok") {
				t.Fatalf("Cluster not ready, output:\n%s", string(b))
			}
		}
	}

	return addrs
}
