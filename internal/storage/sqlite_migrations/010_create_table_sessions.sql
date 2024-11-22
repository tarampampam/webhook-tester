CREATE TABLE IF NOT EXISTS sessions (
  id           VARCHAR(36)      NOT NULL,
  code         UNSIGNED INTEGER NOT NULL DEFAULT 200,
  delay_millis UNSIGNED INTEGER NOT NULL DEFAULT 0,
  body         BLOB             NULL,
  created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at   DATETIME         NOT NULL,
  CONSTRAINT chk_sessions_response_code CHECK (code >= 0 AND code <= 65535)
)
