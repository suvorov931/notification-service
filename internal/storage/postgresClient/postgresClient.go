package postgresClient

import (
	"context"
	"errors"
	"fmt"
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

func New(ctx context.Context, config *Config, metrics monitoring.Monitoring, logger *zap.Logger) (*PostgresService, error) {
	url := buildURL(config)
	dsn := buildDSN(config)

	pool, err := connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	err = upMigration(url)
	if err != nil {
		return nil, err
	}

	return &PostgresService{
		Pool:    pool,
		Metrics: metrics,
		Logger:  logger,
	}, nil
}

func (pr *PostgresService) AddInstantSending(ctx context.Context, email *SMTPClient.EmailMessage) error {
	start := time.Now()

	tag, err := pr.Pool.Exec(ctx, queryForAddInstantSending, email.To, email.Subject, email.Message)
	if err != nil {
		pr.Metrics.Inc("AddInstantSending", monitoring.StatusError)
		pr.Logger.Error("AddInstantSending: failed to add email to database", zap.Error(err))
		return fmt.Errorf("AddInstantSending: failed to add email to database: %w", err)
	}

	duration := time.Since(start).Seconds()
	pr.Metrics.Observe("AddInstantSending", duration)

	if tag.RowsAffected() != 1 {
		pr.Metrics.Inc("AddInstantSending", monitoring.StatusError)
		pr.Logger.Error("AddInstantSending: no rows affected", zap.Error(err))
		return fmt.Errorf("AddInstantSending: no rows affected: %w", err)
	}

	pr.Metrics.Inc("AddInstantSending", monitoring.StatusSuccess)
	return nil
}

func (pr *PostgresService) AddDelayedSending(ctx context.Context, email *SMTPClient.EmailMessageWithTime) error {
	start := time.Now()

	tag, err := pr.Pool.Exec(ctx, queryForAddDelayedSending, email.Time, email.Email.To, email.Email.Subject, email.Email.Message)
	if err != nil {
		pr.Metrics.Inc("AddDelayedSending", monitoring.StatusError)
		pr.Logger.Error("AddDelayedSending: failed to add email to database", zap.Error(err))
		return fmt.Errorf("AddDelayedSending: failed to add email to database: %w", err)
	}

	duration := time.Since(start).Seconds()
	pr.Metrics.Observe("AddDelayedSending", duration)

	if tag.RowsAffected() != 1 {
		pr.Metrics.Inc("AddDelayedSending", monitoring.StatusError)
		pr.Logger.Error("AddDelayedSending: no rows affected", zap.Error(err))
		return fmt.Errorf("AddDelayedSending: no rows affected: %w", err)
	}

	pr.Metrics.Inc("AddDelayedSending", monitoring.StatusSuccess)
	return nil
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

func upMigration(url string) error {
	migration, err := migrate.New("file://./database/migrations", url)
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
