-- CREATE ROLE notification_user WITH LOGIN PASSWORD '1234';

CREATE SCHEMA IF NOT EXISTS schema_emails;

CREATE TABLE IF NOT EXISTS schema_emails.instant_sending
(
    "to" TEXT,
    subject TEXT,
    message TEXT
);

CREATE TABLE IF NOT EXISTS schema_emails.delayed_sending
(
    time INT,
    "to" TEXT,
    subject TEXT,
    message TEXT
);

-- GRANT USAGE ON SCHEMA schema_emails TO notification_user;

-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA schema_emails TO notification_user;

-- ALTER DEFAULT PRIVILEGES IN SCHEMA schema_emails
-- GRANT ALL ON TABLES TO notification_user;
