CREATE TABLE IF NOT EXISTS `response_headers` (
  `sequence`   INTEGER       NOT NULL PRIMARY KEY AUTOINCREMENT,
  `session_id` VARCHAR(36)   NOT NULL,
  `name`       VARCHAR(1024) NOT NULL,
  `value`      TEXT          NOT NULL,
  FOREIGN KEY(`session_id`) REFERENCES `sessions`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
)
