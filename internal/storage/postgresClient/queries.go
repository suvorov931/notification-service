package postgresClient

const (
	queryForAddInstantSending = `INSERT INTO schema_emails.instant_sending ("to", subject,message)
	VALUES ($1, $2, $3) RETURNING id`

	queryForAddDelayedSending = `INSERT INTO schema_emails.delayed_sending (time, "to", subject,message)
	VALUES ($1, $2, $3, $4) RETURNING id`

	queryForFetchById = `WITH found AS (
		SELECT "to", subject, message, NULL::bigint AS time
		FROM schema_emails.instant_sending
		WHERE id = $1

		UNION ALL

		SELECT "to", subject, message, time
		FROM schema_emails.delayed_sending
		WHERE id = $1
	)
	SELECT *
	FROM found
	LIMIT 1;`

	queryForFetchByMail = `WITH found AS (
		SELECT "to", subject, message, NULL::bigint AS time
		FROM schema_emails.instant_sending
		WHERE "to" = $1

		UNION ALL

		SELECT "to", subject, message, time
		FROM schema_emails.delayed_sending
		WHERE "to" = $1
	)
	SELECT *
	FROM found;`

	queryForFetchByAllInstantSending = `SELECT "to", subject, message, NULL::BIGINT as time FROM schema_emails.instant_sending`
	queryForFetchByAllDelayedSending = `SELECT "to", subject, message, time FROM schema_emails.delayed_sending`

	queryForFetchByAll = `WITH found AS (
		SELECT "to", subject, message, NULL::bigint AS time
		FROM schema_emails.instant_sending

		UNION ALL

		SELECT "to", subject, message, time
		FROM schema_emails.delayed_sending
	)
	SELECT *
	FROM found;`
)
