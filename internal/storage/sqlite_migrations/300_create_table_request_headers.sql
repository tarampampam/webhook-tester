CREATE TABLE IF NOT EXISTS `request_headers` (
  `sequence`   INTEGER       NOT NULL PRIMARY KEY AUTOINCREMENT,
  `request_id` VARCHAR(36)   NOT NULL,
  `name`       VARCHAR(1024) NOT NULL,
  `value`      TEXT          NOT NULL,
  FOREIGN KEY(`request_id`) REFERENCES `requests`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
)
