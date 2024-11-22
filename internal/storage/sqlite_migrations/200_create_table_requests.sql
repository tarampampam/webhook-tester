CREATE TABLE IF NOT EXISTS requests (
  id             VARCHAR(36)   NOT NULL,
  session_id     VARCHAR(36)   NOT NULL,
  method         VARCHAR(10)   NOT NULL,
  client_address VARCHAR(39)   NOT NULL,
  url            VARCHAR(4096) NOT NULL,
  payload        BLOB          NULL,
  created_at     DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  sequence       INTEGER       NOT NULL PRIMARY KEY AUTOINCREMENT,
  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE ON UPDATE CASCADE
)
