CREATE SCHEMA IF NOT EXISTS schema_emails;

CREATE SEQUENCE IF NOT EXISTS schema_emails.shared_email_id_seq;

CREATE TABLE IF NOT EXISTS schema_emails.instant_sending
(
    id BIGINT PRIMARY KEY DEFAULT nextval('schema_emails.shared_email_id_seq'),
    "to" TEXT,
    subject TEXT,
    message TEXT
);

CREATE TABLE IF NOT EXISTS schema_emails.delayed_sending
(
    id BIGINT PRIMARY KEY DEFAULT nextval('schema_emails.shared_email_id_seq'),
    time INT,
    "to" TEXT,
    subject TEXT,
    message TEXT
);