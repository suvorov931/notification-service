package postgresClient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/api"
	"notification/internal/monitoring"
)

// TODO: отдельные миграции для каждого тестового постгреса

const pathToTestMigrations = "file://../../../database/migrations"

func TestSaveEmail(t *testing.T) {
	ctx := context.Background()

	testTime := time.Unix(time.Now().Unix(), 0).UTC()

	postgresService := upPostgres("postgres-for-test-SaveEmail", t)

	tests := []struct {
		name      string
		wantEmail *SMTPClient.EmailMessage
		wantId    int
		wantError error
	}{
		{
			name: "success for instant sending",
			wantEmail: &SMTPClient.EmailMessage{
				Type:    api.KeyForInstantSending,
				To:      "to",
				Subject: "instant",
				Message: "message",
			},
			wantId:    1,
			wantError: nil,
		},
		{
			name: "success for instant delayed",
			wantEmail: &SMTPClient.EmailMessage{
				Type:    api.KeyForDelayedSending,
				Time:    &testTime,
				To:      "to",
				Subject: "delayed",
				Message: "message",
			},
			wantId:    2,
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotId, err := postgresService.SaveEmail(ctx, tt.wantEmail)
			require.NoError(t, err)

			q := `SELECT type, time, "to", subject, message FROM schema_emails.emails WHERE subject = $1`
			row := postgresService.pool.QueryRow(ctx, q, tt.wantEmail.Subject)

			var sendingType, to, subject, message string
			var sendingTime *time.Time

			err = row.Scan(&sendingType, &sendingTime, &to, &subject, &message)
			require.NoError(t, err)

			gotEmail := &SMTPClient.EmailMessage{
				Type:    sendingType,
				Time:    sendingTime,
				To:      to,
				Subject: subject,
				Message: message,
			}

			assert.Equal(t, tt.wantId, gotId)
			assert.Equal(t, tt.wantEmail, gotEmail)
		})
	}
}

func upPostgres(name string, t *testing.T) *PostgresService {
	ctx := context.Background()

	t.Helper()

	config := &Config{
		Host:     "localhost",
		User:     "root",
		Password: "test-password",
		Database: "root",
		MaxConns: 5,
		MinConns: 10,
	}

	req := testcontainers.ContainerRequest{
		Name:         name,
		Image:        "postgres:17.5",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     config.User,
			"POSTGRES_PASSWORD": config.Password,
			"POSTGRES_DB":       config.Database,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            false,
	})
	if err != nil {
		t.Fatal(err)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	config.Port = port.Port()

	postgresService, err := New(ctx, config, monitoring.NewNop(), zap.NewNop(), pathToTestMigrations)
	if err != nil {
		t.Fatal(err)
	}

	return postgresService
}
