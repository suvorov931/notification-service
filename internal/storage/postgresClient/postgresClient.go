package postgresClient

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
)

func New(ctx context.Context, config *Config, metrics monitoring.Monitoring, logger *zap.Logger, migrationsPath string) (*PostgresService, error) {
	url := buildURL(config)
	dsn := buildDSN(config)

	pool, err := connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	err = upMigration(url, migrationsPath)
	if err != nil {
		return nil, err
	}

	return &PostgresService{
		pool:    pool,
		metrics: metrics,
		logger:  logger,
	}, nil
}

func (ps *PostgresService) SavingInstantSending(ctx context.Context, email *SMTPClient.EmailMessage) (int, error) {
	start := time.Now()

	var id int

	err := ps.pool.QueryRow(ctx, queryForAddInstantSending,
		email.To, email.Subject, email.Message).
		Scan(&id)
	if err != nil {
		ps.metrics.IncError("SavingInstantSending")
		ps.logger.Error("SavingInstantSending: failed to add email to database", zap.Error(err))
		return 0, fmt.Errorf("SavingInstantSending: failed to add email to database: %w", err)
	}

	ps.metrics.Observe("SavingInstantSending", start)

	ps.metrics.IncSuccess("SavingInstantSending")

	ps.logger.Info(
		"SavingInstantSending: successfully add email to database",
		zap.Any("email", email),
		zap.Int("id", id),
	)

	return id, nil
}

func (ps *PostgresService) SavingDelayedSending(ctx context.Context, email *SMTPClient.EmailMessageWithTime) (int, error) {
	start := time.Now()

	var id int

	err := ps.pool.QueryRow(ctx, queryForAddDelayedSending,
		email.Time, email.Email.To, email.Email.Subject, email.Email.Message).
		Scan(&id)
	if err != nil {
		ps.metrics.IncError("SavingDelayedSending")
		ps.logger.Error("SavingDelayedSending: failed to add email to database", zap.Error(err))
		return 0, fmt.Errorf("SavingDelayedSending: failed to add email to database: %w", err)
	}

	ps.metrics.Observe("SavingDelayedSending", start)

	ps.metrics.IncSuccess("SavingDelayedSending")

	ps.logger.Info(
		"SavingDelayedSending: successfully add email to database",
		zap.Any("email", email),
		zap.Int("id", id),
	)

	return id, nil
}

func (ps *PostgresService) FetchById(ctx context.Context, id string) (any, error) {
	var to, subject, message string
	t := sql.NullInt64{}
	start := time.Now()

	row := ps.pool.QueryRow(ctx, queryForFetchById, id)

	err := row.Scan(&to, &subject, &message, &t)
	if err != nil {
		ps.metrics.IncError("FetchById")
		ps.logger.Error("FetchById: failed to fetch email by id", zap.Error(err))
		return nil, fmt.Errorf("FetchById: failed to fetch email by id: %w", err)
	}

	email := createModel(t, to, subject, message)

	ps.metrics.Observe("FetchById", start)
	ps.metrics.IncSuccess("FetchById")

	ps.logger.Info("FetchById: successfully fetched email", zap.String("id", id))

	return email, nil
}

func createModel(t sql.NullInt64, to, subject, message string) any {
	switch {
	case t.Valid:
		email := &SMTPClient.EmailMessageWithTime{
			Time: strconv.Itoa(int(t.Int64)),
			Email: SMTPClient.EmailMessage{
				To:      to,
				Subject: subject,
				Message: message,
			},
		}

		return email

	default:
		email := &SMTPClient.EmailMessage{
			To:      to,
			Subject: subject,
			Message: message,
		}

		return email
	}
}

func buildURL(config *Config) string {
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	return url
}

func buildDSN(config *Config) string {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s pool_max_conns=%d pool_min_conns=%d",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.MaxConns,
		config.MinConns,
	)

	return dsn
}

func upMigration(url string, path string) error {
	migration, err := migrate.New(path, url)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	err = migration.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	return nil
}

func connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}
