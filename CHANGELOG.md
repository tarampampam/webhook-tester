# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keepachangelog] and this project adheres to [Semantic Versioning][semver].

## UNRELEASED

### Changed

- Go updated from `1.16.3` up to `1.17.5`

## v0.4.0

### Changed

- For redis storage and pubsub `json` encoder/decoder replaced with [msgpack](https://github.com/vmihailenco/msgpack)
- Storage now accepts and returns request content as a bytes slice
- Response content for session creation handler must be base64-encoded
- All API handlers return request body and session response base64-encoded (JSON request and response property `content` replaced with `content_base64`)

### Added

- UI Binary request viewer
- Possibility to download request content using UI

### Fixed

- API handler "all session requests" sorting order

## v0.3.1

### Changed

- JSON request payloads now pretty-printed [#78]

### Fixed

- Request body code highlighting [#77]

[#77]:https://github.com/tarampampam/webhook-tester/issues/77
[#78]:https://github.com/tarampampam/webhook-tester/issues/78

## v0.3.0

### Added

- E2E test using postman
- In-memory storage implementation
- In-memory and Redis pub/sub implementation
- Websockets for browser notifications
- `serve` sub-command flags:
  - `--redis-dsn` redis server DSN (format: `redis://<user>:<password>@<host>:<port>/<db_number>`), required when storage driver `redis` is used
  - `--pubsub-driver` pub/sub driver (`redis` and `memory` is supported)
  - `--storage-driver` storage driver (`redis` and `memory` is supported)
  - `--max-request-body-size` maximal webhook request body size (in bytes)
  - `--ws-max-clients` maximal websocket clients
  - `--ws-max-lifetime` maximal single websocket lifetime
- Sub-command `healthcheck` (hidden in CLI help) that makes a simple HTTP request (with user-agent `HealthChecker/internal`) to the `http://127.0.0.1:8080/live` endpoint. Port number can be changed using `--port`, `-p` flag or `LISTEN_PORT` environment variable
- Healthcheck in dockerfile
- Global (available for all sub-commands) flags:
  - `--log-json` for logging using JSON format (`stderr`)
  - `--debug` for debug information for logging messages
  - `--verbose` for verbose output
- Graceful shutdown support for `serve` sub-command
- HTTP requests & HTTP panics logging middlewares
- Logging using `uber-go/zap` package
- HTTP route `/api/version`
- HTTP route `/metrics` with metrics in [prometheus](https://github.com/prometheus) format
- Support for `linux/arm64`, `linux/arm/v6` and `linux/arm/v7` platforms for docker image

### Changed

- Docker image based on `scratch` (not `alpine` image)
- Directory `public` renamed to `web`
- Package name changed from `webhook-tester` to `github.com/tarampampam/webhook-tester`
- Default value for `--public` flag (`serve` sub-command) now `%binary_file_dir%/web` instead `%current_working_directory%/web`
- Flag `--session-ttl` (`serve` sub-command) now accepts duration (example: `1h30m`) instead seconds count
- Flag `--public` now accepts empty value (in this case file server will be disabled)
- Go updated from `1.15` up to `1.16.3`
- All frontend libraries updated

### Removed

- Binary file packing using `upx`
- HTTP method `CONNECT` for webhook endpoint
- `serve` sub-command flags:`--redis-host`, `--redis-port`, `--redis-db-num`
- Environment variables support: `REDIS_HOST`, `REDIS_PORT`, `REDIS_DB_NUM`
- Property `version` from `/api/settings` JSON-object response
- Pusher support (replaced with pub/sub + websockets)

### Fixed

- Webhook delay uses `time.After()` instead `time.Sleep()` with context canceling support
- Wrong `content` property value for session creation HTTP handler
- Wrong `Content-Type` header value for webhook handler
- Correct file mime-types for docker image

## v0.2.0

### Changed

- Golang updated from `1.14` up to `1.15`

## v0.1.0

### Added

- Flag `--ignore-header-prefix` for incoming webhook headers ignoring (by prefix)

## v0.0.1

### Added

- Basic features, like UI, `pusher.com`, probes, session and request handlers

[keepachangelog]:https://keepachangelog.com/en/1.0.0/
[semver]:https://semver.org/spec/v2.0.0.html
