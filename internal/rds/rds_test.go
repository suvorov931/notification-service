package rds

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

const passForRedisTestContainer = "something password"

// TODO: проверить аутентификацию для редиса (NOAUTH, WRONGPASS)?

func TestAddDelayedEmail(t *testing.T) {
	ctx := context.Background()

	container, _, _ := upRedisContainer(ctx, t)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("cannot get host: %v", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("cannot get port: %v", err)
	}

	cfg := &Config{
		Addr:     fmt.Sprintf("%s:%s", host, port.Port()),
		Password: passForRedisTestContainer,
	}

	client, err := New(ctx, cfg, zap.NewNop())
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(client)
	//tests := []struct {
	//	name      string
	//	email     *service.EmailWithTime
	//	wantEmail []string
	//}{
	//	{
	//		name:      "success add",
	//		email:     &service.EmailWithTime{},
	//		wantEmail: []string{"{\"Time\":\"1764687845\",\"Email\":{\"to\":\"daanisimov04@gmail.com\",\"subject\":\"subject\",\"message\":\"message\"}}"},
	//	},
	//}
	//
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//
	//		res, err := rdsCl.client.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
	//			Min: "1764687845",
	//			Max: "1764687845",
	//		}).Result()
	//		if err != nil {
	//			t.Error(err.Error())
	//		}
	//
	//		if !reflect.DeepEqual(res, tt.wantEmail) {
	//			t.Errorf("AddDelayedEmail(): email = %v, wantEmail = %s", res, tt.wantEmail)
	//		}
	//
	//	})
	//}

}

func upRedisContainer(ctx context.Context, t *testing.T) (testcontainers.Container, string, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Name:         "redis-for-test",
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
	if err != nil {
		t.Fatalf("upRedisContainer: failed to start container %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("upRedisContainer: cannot get host: %v", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("upRedisContainer: cannot get port: %v", err)
	}

	return container, host, port.Port()
}
