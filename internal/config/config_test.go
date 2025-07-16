package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := tempDir + "/config.env"

	content := `
	HTTP_HOST=localhost
	HTTP_PORT=8080
	HTTP_MONITORING_PORT=2112

	SENDER_EMAIL=something@mail.ru
	SENDER_PASSWORD=somethingPassword
	SMTP_HOST=hostForSMTP
	SMTP_PORT=12345
	SKIP_VERIFY=false
	MAX_RETRIES=3
	BASIC_RETRY_PAUSE=3s

	REDIS_CLUSTER_ADDRS=redis-node-1:7001,redis-node-2:7002,redis-node-3:7003,redis-node-4:7004,redis-node-5:7005,redis-node-6:7006
	REDIS_CLUSTER_TIMEOUT=3s
	REDIS_CLUSTER_PASSWORD=redisPassword
	REDIS_CLUSTER_READ_ONLY=true

	POSTGRES_HOST=localhost
	POSTGRES_PORT=5432
	POSTGRES_USER=root
	POSTGRES_PASSWORD=postgresPassword
	POSTGRES_DATABASE=postgres
	POSTGRES_TIMEOUT=3s
	POSTGRES_MAX_CONNECTIONS=10
	POSTGRES_MIN_CONNECTIONS=5

	LOGGER=dev
	`

	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := New(tempFile)
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.HttpServer.Host)
	assert.Equal(t, "8080", cfg.HttpServer.Port)
	assert.Equal(t, "2112", cfg.HttpServer.MonitoringPort)

	assert.Equal(t, "something@mail.ru", cfg.SMTP.SenderEmail)
	assert.Equal(t, "somethingPassword", cfg.SMTP.SenderPassword)
	assert.Equal(t, "hostForSMTP", cfg.SMTP.SMTPHost)
	assert.Equal(t, 12345, cfg.SMTP.SMTPPort)
	assert.Equal(t, false, cfg.SMTP.SkipVerify)
	assert.Equal(t, 3, cfg.SMTP.MaxRetries)
	assert.Equal(t, 3*time.Second, cfg.SMTP.BasicRetryPause)

	assert.Equal(t, []string{
		"redis-node-1:7001",
		"redis-node-2:7002",
		"redis-node-3:7003",
		"redis-node-4:7004",
		"redis-node-5:7005",
		"redis-node-6:7006",
	}, cfg.Redis.Addrs)
	assert.Equal(t, 3*time.Second, cfg.Redis.Timeout)
	assert.Equal(t, "redisPassword", cfg.Redis.Password)
	assert.Equal(t, true, cfg.Redis.ReadOnly)

	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, "5432", cfg.Postgres.Port)
	assert.Equal(t, "root", cfg.Postgres.User)
	assert.Equal(t, "postgresPassword", cfg.Postgres.Password)
	assert.Equal(t, "postgres", cfg.Postgres.Database)
	assert.Equal(t, 3*time.Second, cfg.Postgres.Timeout)
	assert.Equal(t, 10, cfg.Postgres.MaxConns)
	assert.Equal(t, 5, cfg.Postgres.MinConns)

	assert.Equal(t, "dev", cfg.Logger.Env)

	assert.Equal(t, 3*time.Second, cfg.AppTimeouts.SMTPPauseForRetries)
	assert.Equal(t, 3, cfg.AppTimeouts.SMTPQuantityOfRetries)
	assert.Equal(t, 3*time.Second, cfg.AppTimeouts.RedisTimeout)
	assert.Equal(t, 3*time.Second, cfg.AppTimeouts.PostgresTimeout)

	_, err = New("wrongPath")
	assert.Contains(t, err.Error(), "failed to read config")
}
