package postgresClient

const (
	// queryForSaveEmail inserts a new email into the database and returns its ID.
	queryForSaveEmail = `INSERT INTO schema_emails.emails (type, time, "to", subject, message)
	VALUES ($1, $2, $3, $4, $5) RETURNING id`

	// queryForFetchById selects a single email by its ID.
	queryForFetchById = `SELECT type, time, "to", subject, message FROM schema_emails.emails WHERE id = $1`

	// queryForFetchByEmail selects all emails sent to a specific recipient.
	queryForFetchByEmail = `SELECT type, time, "to", subject, message FROM schema_emails.emails WHERE "to" = $1`

	// queryForFetchByAll selects all emails from the table.
	queryForFetchByAll = `SELECT type, time, "to", subject, message FROM schema_emails.emails`
)
