//package postgresClient
//
//import (
//	"context"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	"github.com/testcontainers/testcontainers-go"
//	"github.com/testcontainers/testcontainers-go/wait"
//	"go.uber.org/zap"
//
//	"notification/internal/SMTPClient"
//	"notification/internal/monitoring"
//)
//
//const pathToTestMigrations = "file://../../../database/migrations"
//
//func TestAddInstantSending(t *testing.T) {
//	ctx := context.Background()
//
//	postgresService := upPostgres("postgres-for-test-TeAddInstantSending", t)
//
//	tests := []struct {
//		name      string
//		email     *SMTPClient.EmailMessage
//		wantId    int
//		wantError error
//	}{
//		{
//			name: "success",
//			email: &SMTPClient.EmailMessage{
//				To:      "to",
//				Subject: "subject",
//				Message: "message",
//			},
//			wantId:    1,
//			wantError: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//
//			gotId, err := postgresService.SavingInstantSending(ctx, tt.email)
//			require.NoError(t, err)
//
//			q := `SELECT "to", subject, message FROM schema_emails.instant_sending WHERE subject = $1`
//			row := postgresService.pool.QueryRow(ctx, q, tt.email.Subject)
//
//			var to, subject, message string
//			err = row.Scan(&to, &subject, &message)
//			require.NoError(t, err)
//
//			gotEmail := &SMTPClient.EmailMessage{
//				To:      to,
//				Subject: subject,
//				Message: message,
//			}
//
//			assert.Equal(t, tt.wantId, gotId)
//			assert.Equal(t, tt.email, gotEmail)
//		})
//	}
//}
//
//func TestAddDelayedSending(t *testing.T) {
//	ctx := context.Background()
//
//	postgresService := upPostgres("postgres-for-test-SavingDelayedSending", t)
//
//	tests := []struct {
//		name      string
//		email     *SMTPClient.EmailMessageWithTime
//		wantId    int
//		wantError error
//	}{
//		{
//			name: "success",
//			email: &SMTPClient.EmailMessageWithTime{
//				Time: "1242421312",
//				Email: SMTPClient.EmailMessage{
//					To:      "to",
//					Subject: "subject",
//					Message: "message",
//				},
//			},
//			wantId:    1,
//			wantError: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//
//			gotId, err := postgresService.SavingDelayedSending(ctx, tt.email)
//			require.NoError(t, err)
//
//			q := `SELECT time, "to", subject, message FROM schema_emails.delayed_sending WHERE subject = $1`
//			row := postgresService.pool.QueryRow(ctx, q, tt.email.Email.Subject)
//
//			var time, to, subject, message string
//			err = row.Scan(&time, &to, &subject, &message)
//			require.NoError(t, err)
//
//			gotEmail := &SMTPClient.EmailMessageWithTime{
//				Time: time,
//				Email: SMTPClient.EmailMessage{
//					To:      to,
//					Subject: subject,
//					Message: message,
//				},
//			}
//
//			assert.Equal(t, tt.email, gotEmail)
//			assert.Equal(t, tt.wantId, gotId)
//		})
//	}
//}
//
//func upPostgres(name string, t *testing.T) *PostgresService {
//	ctx := context.Background()
//
//	t.Helper()
//
//	config := &Config{
//		Host:     "localhost",
//		User:     "root",
//		Password: "test-password",
//		Database: "root",
//		MaxConns: 5,
//		MinConns: 10,
//	}
//
//	req := testcontainers.ContainerRequest{
//		Name:         name,
//		Image:        "postgres:17.5",
//		ExposedPorts: []string{"5432/tcp"},
//		Env: map[string]string{
//			"POSTGRES_USER":     config.User,
//			"POSTGRES_PASSWORD": config.Password,
//			"POSTGRES_DB":       config.Database,
//		},
//		WaitingFor: wait.ForListeningPort("5432/tcp"),
//	}
//
//	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
//		ContainerRequest: req,
//		Started:          true,
//		Reuse:            false,
//	})
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	port, err := container.MappedPort(ctx, "5432/tcp")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	config.Port = port.Port()
//
//	postgresService, err := New(ctx, config, monitoring.NewNop(), zap.NewNop(), pathToTestMigrations)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	return postgresService
//}

package postgresClient
