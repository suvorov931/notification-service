package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := tempDir + "/config.yaml"

	content := `
HTTP_SERVER:
  HTTP_HOST: localhost
  HTTP_PORT: 8080
SMTP:
  SENDER_EMAIL: something@mail.ru
  SENDER_PASSWORD: somethingPassword
  SMTP_HOST: hostForSMTP
  SMTP_PORT: 12345
  SKIP_VERIFY: false
  MAX_RETRIES: 3
  BASIC_RETRY_PAUSE: 5
REDIS:
  REDIS_ADDR: localhost:6379
  REDIS_PASSWORD: 12345
  REDIS_DB: 
  REDIS_USERNAME:
LOGGER:
  env: dev
`

	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := New(tempFile)
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.HttpServer.Host)
	assert.Equal(t, "8080", cfg.HttpServer.Port)

	assert.Equal(t, "something@mail.ru", cfg.SMTP.SenderEmail)
	assert.Equal(t, "somethingPassword", cfg.SMTP.SenderPassword)
	assert.Equal(t, "hostForSMTP", cfg.SMTP.SMTPHost)
	assert.Equal(t, 12345, cfg.SMTP.SMTPPort)
	assert.Equal(t, false, cfg.SMTP.SkipVerify)
	assert.Equal(t, 3, cfg.SMTP.MaxRetries)
	assert.Equal(t, 5, cfg.SMTP.BasicRetryPause)

	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Equal(t, "12345", cfg.Redis.Password)

	assert.Equal(t, "dev", cfg.Logger.Env)

	_, err = New("wrongPath")
	assert.Contains(t, err.Error(), "failed to read config")
}
