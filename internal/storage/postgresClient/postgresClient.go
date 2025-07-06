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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/api"
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
		pool:    pool,
		metrics: metrics,
		logger:  logger,
	}, nil
}

func (pr *PostgresService) AddSending(ctx context.Context, key string, email any) error {
	if ctx.Err() != nil {
		pr.metrics.Inc("AddSending", monitoring.StatusCanceled)
		pr.logger.Warn("AddSending: context canceled before adding email ti postgres", zap.Error(ctx.Err()))
		return fmt.Errorf("AddSending: context canceled before adding email ti postgres")
	}

	start := time.Now()

	var err error
	var tag pgconn.CommandTag

	switch key {
	case api.KeyForInstantSending:
		e := email.(*SMTPClient.EmailMessage)
		tag, err = pr.pool.Exec(ctx, queryForAddInstantSending, e.To, e.Subject, e.Message)
	case api.KeyForDelayedSending:
		e := email.(*SMTPClient.EmailMessageWithTime)
		tag, err = pr.pool.Exec(ctx, queryForAddDelayedSending, e.Time, e.Email.To, e.Email.Subject, e.Email.Message)
	}

	if err != nil {
		pr.metrics.Inc("AddSending", monitoring.StatusError)
		pr.logger.Error("AddSending: failed to add email to database", zap.Error(err))
		return fmt.Errorf("AddSending: failed to add email to database: %w", err)
	}

	duration := time.Since(start).Seconds()
	pr.metrics.Observe("AddSending", duration)

	if tag.RowsAffected() != 1 {
		pr.metrics.Inc("AddSending", monitoring.StatusError)
		pr.logger.Error("AddSending: no rows affected", zap.Error(err))
		return fmt.Errorf("AddSending: no rows affected: %w", err)
	}

	pr.metrics.Inc("AddSending", monitoring.StatusSuccess)
	pr.logger.Info("AddSending: successfully add email to database", zap.Any("email", email))
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
