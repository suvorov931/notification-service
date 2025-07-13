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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
)

func New(ctx context.Context, config *Config, metrics monitoring.Monitoring, logger *zap.Logger, migrationsPath string) (*PostgresService, error) {
	if config.Timeout == 0 {
		config.Timeout = DefaultPostgresTimeout
	}

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
		timeout: config.Timeout,
	}, nil
}

func (ps *PostgresService) SaveEmail(ctx context.Context, email *SMTPClient.EmailMessage) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, ps.timeout)
	defer cancel()

	start := time.Now()

	var id int

	err := ps.pool.QueryRow(ctx, queryForSaveEmail,
		email.Type, email.Time, email.To, email.Subject, email.Message).Scan(&id)

	if err != nil {
		return 0, ps.processContextError("SaveEmail", err)
	}

	ps.metrics.Observe("SaveEmail", start)

	ps.metrics.IncSuccess("SaveEmail")

	ps.logger.Info(
		"SaveEmail: successfully add email to database",
		zap.Any("email", email),
		zap.Int("id", id),
	)

	return id, nil
}

func (ps *PostgresService) FetchById(ctx context.Context, id int) ([]*SMTPClient.EmailMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, ps.timeout)
	defer cancel()

	start := time.Now()

	var sendingType, to, subject, message string
	var sendingTime *time.Time

	row := ps.pool.QueryRow(ctx, queryForFetchById, id)

	err := row.Scan(&sendingType, &sendingTime, &to, &subject, &message)
	if err != nil {
		return nil, ps.processContextError("FetchById", err)
	}

	res := &SMTPClient.EmailMessage{
		Type:    sendingType,
		Time:    sendingTime,
		To:      to,
		Subject: subject,
		Message: message,
	}

	ps.metrics.Observe("FetchById", start)
	ps.metrics.IncSuccess("FetchById")

	ps.logger.Info("FetchById: successfully fetched email by id", zap.Int("id", id))

	return []*SMTPClient.EmailMessage{res}, nil
}

func (ps *PostgresService) FetchByEmail(ctx context.Context, email string) ([]*SMTPClient.EmailMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, ps.timeout)
	defer cancel()

	start := time.Now()
	fmt.Println(email)
	rows, err := ps.pool.Query(ctx, queryForFetchByEmail, email)
	if err != nil {
		return nil, ps.processContextError("FetchByMail", err)
	}

	defer rows.Close()

	res, err := ps.processRows(rows)
	if err != nil {
		return nil, err
	}

	ps.metrics.Observe("FetchByMail", start)
	ps.metrics.IncSuccess("FetchByMail")

	ps.logger.Info("FetchByMail: successfully fetched email by mail", zap.String("mail", email))

	return res, nil
}

func (ps *PostgresService) FetchByAll(ctx context.Context) ([]*SMTPClient.EmailMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, ps.timeout)
	defer cancel()

	start := time.Now()

	rows, err := ps.pool.Query(ctx, queryForFetchByAll)
	if err != nil {
		return nil, ps.processContextError("FetchByAll", err)
	}

	defer rows.Close()

	res, err := ps.processRows(rows)
	if err != nil {
		return nil, err
	}

	ps.metrics.Observe("FetchByAll", start)
	ps.metrics.IncSuccess("FetchByAll")

	ps.logger.Info("FetchByAll: successfully fetched email by al")

	return res, nil
}

func (ps *PostgresService) Close() {
	ps.pool.Close()
}

func (ps *PostgresService) processContextError(funcName string, err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		ps.metrics.IncCanceled(funcName)
		ps.logger.Error(fmt.Sprintf("%s: context canceled", funcName), zap.Error(err))

		return fmt.Errorf("%s: context canceled: %w", funcName, err)

	case errors.Is(err, context.DeadlineExceeded):
		ps.metrics.IncTimeout(funcName)
		ps.logger.Error(fmt.Sprintf("%s: deadline context", funcName), zap.Error(err))

		return fmt.Errorf("%s: deadline context: %w", funcName, err)

	default:
		ps.metrics.IncError(funcName)
		ps.logger.Error(funcName, zap.Error(err))

		return fmt.Errorf("%s: %w", funcName, err)
	}
}

func (ps *PostgresService) processRows(rows pgx.Rows) ([]*SMTPClient.EmailMessage, error) {
	var emails []*SMTPClient.EmailMessage

	for rows.Next() {
		var sendingType, to, subject, message string
		var sendingTime *time.Time

		err := rows.Scan(&sendingType, &sendingTime, &to, &subject, &message)
		if err != nil {
			ps.metrics.IncError("processRows")
			ps.logger.Error("processRows: failed to fetch email", zap.Error(err))
			return nil, fmt.Errorf("processRows: failed to fetch email: %w", err)
		}

		email := &SMTPClient.EmailMessage{
			Type:    sendingType,
			Time:    sendingTime,
			To:      to,
			Subject: subject,
			Message: message,
		}

		emails = append(emails, email)
	}

	if rows.Err() != nil {
		ps.metrics.IncError("processRows")
		ps.logger.Error("processRows: rows error", zap.Error(rows.Err()))
		return nil, fmt.Errorf("processRows: rows error: %w", rows.Err())
	}

	return emails, nil
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
