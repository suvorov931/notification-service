package postgresClient

const (
	queryForSaveEmail = `INSERT INTO schema_emails.emails (type, time, "to", subject, message)
	VALUES ($1, $2, $3, $4, $5) RETURNING id`

	queryForFetchById = `SELECT type, time, "to", subject, message FROM schema_emails.emails WHERE id = $1`

	queryForFetchByEmail = `SELECT type, time, "to", subject, message FROM schema_emails.emails WHERE "to" = $1`

	queryForFetchByAll = `SELECT type, time, "to", subject, message FROM schema_emails.emails`
)
