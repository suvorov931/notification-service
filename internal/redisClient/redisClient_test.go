package redisClient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

	"notification/internal/notification/SMTPClient"
	"notification/internal/notification/api"
)

func TestAddDelayedEmail(t *testing.T) {
	ctx := context.Background()

	addrs := upRedisCluster(ctx, "TestAddDelayedEmail", 1, t)

	rc, err := New(ctx, &Config{Addrs: addrs}, zap.NewNop())
	require.NoError(t, err)

	tests := []struct {
		name      string
		email     *SMTPClient.EmailMessageWithTime
		wantEmail []string
		wantErr   error
	}{
		{
			name: "success add",
			email: &SMTPClient.EmailMessageWithTime{
				Time: "2025-12-02 15:04:05",
				Email: SMTPClient.EmailMessage{
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

			email, err := rc.Cluster.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
				Min: tt.email.Time,
				Max: tt.email.Time,
			}).Result()
			require.NoError(t, err)

			assert.Equal(t, tt.wantEmail, email)
		})
	}

}

func TestAddDelayedEmailTimeout(t *testing.T) {
	ctx := context.Background()
	db, rdsMock := redismock.NewClusterMock()

	rc := RedisCluster{
		Cluster: db,
		Logger:  zap.NewNop(),
	}

	email := &SMTPClient.EmailMessageWithTime{
		Time: "2026-01-01 01:01:01",
		Email: SMTPClient.EmailMessage{
			To:      "daanisimov04@gmail.com",
			Subject: "test",
			Message: "message",
		},
	}

	parseEmail := `{"Time":"1767229261","Email":{"to":"daanisimov04@gmail.com","subject":"test","message":"message"}}`

	rdsMock.ExpectZAdd(api.KeyForDelayedSending, redis.Z{
		Score:  float64(1767229261),
		Member: []byte(parseEmail),
	}).SetErr(context.DeadlineExceeded)

	err := rc.AddDelayedEmail(ctx, email)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCheckRedis(t *testing.T) {
	ctx := context.Background()

	addrs := upRedisCluster(ctx, "TestCheckRedis", 2, t)

	rc, err := New(ctx, &Config{Addrs: addrs}, zap.NewNop())
	require.NoError(t, err)

	tests := []struct {
		name    string
		z       []redis.Z
		want    []string
		wantErr error
		delFunc func(rc redis.ClusterClient)
	}{
		{
			name: "success check one entry",
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "1"},
			},
			want:    []string{"1"},
			wantErr: nil,
			delFunc: func(rc redis.ClusterClient) {
				rc.ZRem(ctx, api.KeyForDelayedSending, "1")
			},
		},
		{
			name: "success check two entry",
			z: []redis.Z{
				{Score: float64(time.Now().Unix()), Member: "2"},
				{Score: float64(time.Now().Unix()), Member: "22"},
			},
			want:    []string{"2", "22"},
			wantErr: nil,
			delFunc: func(rc redis.ClusterClient) {
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
			delFunc: func(rc redis.ClusterClient) {
				rc.ZRem(ctx, api.KeyForDelayedSending)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = rc.Cluster.ZAdd(ctx, api.KeyForDelayedSending, tt.z...).Err()
			require.NoError(t, err)

			res, err := rc.CheckRedis(ctx)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, res)

			tt.delFunc(*rc.Cluster)
		})
	}

	t.Run("check removal after reading", func(t *testing.T) {
		err := rc.Cluster.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: "something",
		}).Err()
		require.NoError(t, err)

		res, err := rc.CheckRedis(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"something"}, res)

		emptyRes, err := rc.Cluster.ZRange(ctx, api.KeyForDelayedSending, 0, -1).Result()
		require.NoError(t, err)
		assert.Equal(t, []string{}, emptyRes)

	})
}

func TestCheckRedisTimeout(t *testing.T) {
	ctx := context.Background()
	db, rdsMock := redismock.NewClusterMock()

	rc := RedisCluster{
		Cluster: db,
		Logger:  zap.NewNop(),
	}

	now := float64(time.Now().Unix())

	rdsMock.ExpectZRangeByScore(api.KeyForDelayedSending, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(now, 'f', -1, 64),
	}).SetErr(context.DeadlineExceeded)

	res, err := rc.CheckRedis(ctx)

	assert.Nil(t, res)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

}

func TestFailedConnection(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		Addrs: []string{
			"localhost:1234",
		},
		Password: "wrong",
	}

	_, err := New(ctx, cfg, zap.NewNop())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}

func TestParseAndConvertTime(t *testing.T) {
	rc := RedisCluster{
		Logger: zap.NewNop(),
	}

	tests := []struct {
		name        string
		email       *SMTPClient.EmailMessageWithTime
		expectedErr bool
	}{
		{
			name: "success",
			email: &SMTPClient.EmailMessageWithTime{
				Time: "2035-06-27 15:04:05",
				Email: SMTPClient.EmailMessage{
					To:      "test@gmail.com",
					Subject: "subject",
					Message: "message",
				},
			},
			expectedErr: false,
		},
		{
			name: "invalid time",
			email: &SMTPClient.EmailMessageWithTime{
				Time: "invalid time",
				Email: SMTPClient.EmailMessage{
					To:      "test@gmail.com",
					Subject: "subject",
					Message: "message",
				},
			},
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, score, err := rc.parseAndConvertTime(tt.email)

			if tt.expectedErr {
				require.Error(t, err)
				assert.Nil(t, res)
				assert.Equal(t, 0.0, score)
				assert.Contains(t, err.Error(), "parseAndConvertTime: cannot parse email.Time")
			} else {
				require.NoError(t, err)

				wantScore, err := time.ParseInLocation("2006-01-02 15:04:05", "2035-06-27 15:04:05", time.UTC)
				require.NoError(t, err)

				assert.Equal(t, float64(wantScore.Unix()), score)

				var wantEmail SMTPClient.EmailMessageWithTime
				err = json.Unmarshal(res, &wantEmail)
				require.NoError(t, err)
				assert.Equal(t, wantEmail.Email, tt.email.Email)
			}
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
		}

	}

	return addrs
}
