package postgresClient

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/api"
	"notification/internal/monitoring"
)

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

func TestFetchById(t *testing.T) {
	ctx := context.Background()

	postgresService := upPostgres("postgres-for-test-FetchById", t)

	insertData := func(insertSQL string) {
		_, err := postgresService.pool.Exec(ctx, insertSQL)
		require.NoError(t, err)
	}

	clearData := func() {
		_, err := postgresService.pool.Exec(ctx, `DELETE FROM schema_emails.emails`)
		require.NoError(t, err)
	}

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2035-07-13 21:58:00", time.UTC)
	require.NoError(t, err)

	tests := []struct {
		name      string
		setup     func(insertSQL string)
		insertSQL string
		id        int
		want      []*SMTPClient.EmailMessage
		wantErr   error
	}{
		{
			name: "success for instant sending",
			setup: func(insertSQL string) {
				clearData()
				insertData(insertSQL)
			},
			insertSQL: `INSERT INTO schema_emails.emails (id ,type, time, "to", subject, message) VALUES
			(1,'instantSending', null, 'to', 'subject', 'message');`,
			id: 1,
			want: []*SMTPClient.EmailMessage{{
				Type:    api.KeyForInstantSending,
				To:      "to",
				Subject: "subject",
				Message: "message",
			}},
			wantErr: nil,
		},
		{
			name: "success for delayed sending",
			setup: func(insertSQL string) {
				clearData()
				insertData(insertSQL)
			},
			insertSQL: `INSERT INTO schema_emails.emails (id ,type, time, "to", subject, message) VALUES
			(2,'delayedSending', '2035-07-13 21:58:00', 'to', 'subject', 'message');`,
			id: 2,
			want: []*SMTPClient.EmailMessage{{
				Type:    api.KeyForDelayedSending,
				Time:    &testTime,
				To:      "to",
				Subject: "subject",
				Message: "message",
			}},
			wantErr: nil,
		},
		{
			name: "id not exists",
			setup: func(insertSQL string) {
				clearData()
			},
			id:      0,
			want:    nil,
			wantErr: pgx.ErrNoRows,
		},
		{
			name: "negative id",
			setup: func(insertSQL string) {
				clearData()
			},
			id:      -1,
			want:    nil,
			wantErr: pgx.ErrNoRows,
		},
		{
			name: "large id",
			setup: func(insertSQL string) {
				clearData()
			},
			id:      math.MaxInt32,
			want:    nil,
			wantErr: pgx.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.insertSQL)

			got, err := postgresService.FetchById(ctx, tt.id)

			assert.Equal(t, tt.want, got)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestFetchByEmail(t *testing.T) {
	ctx := context.Background()

	postgresService := upPostgres("postgres-for-test-FetchByEmail", t)

	insertData := func(insertSQL string) {
		_, err := postgresService.pool.Exec(ctx, insertSQL)
		require.NoError(t, err)
	}

	clearData := func() {
		_, err := postgresService.pool.Exec(ctx, `DELETE FROM schema_emails.emails`)
		require.NoError(t, err)
	}

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2035-07-13 21:58:00", time.UTC)
	require.NoError(t, err)

	tests := []struct {
		name      string
		setup     func(insertSQL string)
		insertSQL string
		email     string
		want      []*SMTPClient.EmailMessage
		wantErr   error
	}{
		{
			name: "success for instant sending",
			setup: func(insertSQL string) {
				clearData()
				insertData(insertSQL)
			},
			insertSQL: `INSERT INTO schema_emails.emails (id ,type, time, "to", subject, message) VALUES 
        	(1,'instantSending', null, 'to', 'subject', 'message');`,
			email: "to",
			want: []*SMTPClient.EmailMessage{{
				Type:    api.KeyForInstantSending,
				To:      "to",
				Subject: "subject",
				Message: "message",
			}},
			wantErr: nil,
		},
		{
			name:  "success for delayed sending",
			email: "to1",
			setup: func(insertSQL string) {
				clearData()
				insertData(insertSQL)
			},
			insertSQL: `INSERT INTO schema_emails.emails (id ,type, time, "to", subject, message) VALUES 
        	(2,'delayedSending', '2035-07-13 21:58:00', 'to1', 'subject', 'message');`,
			want: []*SMTPClient.EmailMessage{{
				Type:    api.KeyForDelayedSending,
				Time:    &testTime,
				To:      "to1",
				Subject: "subject",
				Message: "message",
			}},
			wantErr: nil,
		},
		{
			name:  "multiple entry",
			email: "common",
			setup: func(insertSQL string) {
				clearData()
				insertData(insertSQL)
			},
			insertSQL: `INSERT INTO schema_emails.emails (id ,type, time, "to", subject, message) VALUES 
			(1,'instantSending', null, 'common', 'subject', 'message'),
        	(2,'delayedSending', '2035-07-13 21:58:00', 'common', 'subject', 'message');`,
			want: []*SMTPClient.EmailMessage{
				{
					Type:    api.KeyForInstantSending,
					To:      "common",
					Subject: "subject",
					Message: "message",
				},
				{
					Type:    api.KeyForDelayedSending,
					Time:    &testTime,
					To:      "common",
					Subject: "subject",
					Message: "message",
				},
			},
			wantErr: nil,
		},
		{
			name: "email not exists",
			setup: func(insertSQL string) {
				clearData()
			},
			email:   "somethingEmail",
			want:    nil,
			wantErr: pgx.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.insertSQL)

			got, err := postgresService.FetchByEmail(ctx, tt.email)

			assert.Equal(t, tt.want, got)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestFetchByAll(t *testing.T) {
	ctx := context.Background()

	postgresService := upPostgres("postgres-for-test-FetchByAll", t)

	insertData := func(insertSQL string) {
		_, err := postgresService.pool.Exec(ctx, insertSQL)
		require.NoError(t, err)
	}

	clearData := func() {
		_, err := postgresService.pool.Exec(ctx, `DELETE FROM schema_emails.emails`)
		require.NoError(t, err)
	}

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2035-07-13 21:58:00", time.UTC)
	require.NoError(t, err)

	tests := []struct {
		name      string
		setup     func(insertSQL string)
		insertSQL string
		want      []*SMTPClient.EmailMessage
		wantErr   error
	}{
		{
			name: "success",
			setup: func(insertSQL string) {
				clearData()
				insertData(insertSQL)
			},
			insertSQL: `INSERT INTO schema_emails.emails (id ,type, time, "to", subject, message) VALUES
			(1,'instantSending', null, 'to', 'subject', 'message'),
			(2,'delayedSending', '2035-07-13 21:58:00', 'to', 'subject', 'message');`,
			want: []*SMTPClient.EmailMessage{
				{
					Type:    api.KeyForInstantSending,
					To:      "to",
					Subject: "subject",
					Message: "message",
				},
				{
					Type:    api.KeyForDelayedSending,
					Time:    &testTime,
					To:      "to",
					Subject: "subject",
					Message: "message",
				},
			},
			wantErr: nil,
		},
		{
			name: "entry not exists",
			setup: func(insertSQL string) {
				clearData()
			},
			want:    nil,
			wantErr: pgx.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.insertSQL)

			got, err := postgresService.FetchByAll(ctx)

			assert.Equal(t, tt.want, got)
			assert.ErrorIs(t, err, tt.wantErr)
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
