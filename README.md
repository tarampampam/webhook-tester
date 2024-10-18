<!--GENERATED:CLI_DOCS-->
<!-- Documentation inside this block generated by github.com/urfave/cli; DO NOT EDIT -->
## CLI interface

webhook tester.

Usage:

```bash
$ app [GLOBAL FLAGS] [COMMAND] [COMMAND FLAGS] [ARGUMENTS...]
```

Global flags:

| Name               | Description                                 | Default value | Environment variables |
|--------------------|---------------------------------------------|:-------------:|:---------------------:|
| `--log-level="…"`  | Logging level (debug/info/warn/error/fatal) |    `info`     |      `LOG_LEVEL`      |
| `--log-format="…"` | Logging format (console/json)               |   `console`   |     `LOG_FORMAT`      |

### `start` command (aliases: `s`, `server`, `serve`, `http-server`)

Start HTTP/HTTPs servers.

Usage:

```bash
$ app [GLOBAL FLAGS] start [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name                          | Description                                                                                                               |       Default value        |    Environment variables     |
|-------------------------------|---------------------------------------------------------------------------------------------------------------------------|:--------------------------:|:----------------------------:|
| `--addr="…"`                  | IP (v4 or v6) address to listen on (0.0.0.0 to bind to all interfaces)                                                    |         `0.0.0.0`          | `SERVER_ADDR`, `LISTEN_ADDR` |
| `--http-port="…"`             | HTTP server port                                                                                                          |           `8080`           |         `HTTP_PORT`          |
| `--read-timeout="…"`          | maximum duration for reading the entire request, including the body (zero = no timeout)                                   |           `1m0s`           |     `HTTP_READ_TIMEOUT`      |
| `--write-timeout="…"`         | maximum duration before timing out writes of the response (zero = no timeout)                                             |           `1m0s`           |     `HTTP_WRITE_TIMEOUT`     |
| `--idle-timeout="…"`          | maximum amount of time to wait for the next request (keep-alive, zero = no timeout)                                       |           `1m0s`           |     `HTTP_IDLE_TIMEOUT`      |
| `--storage-driver="…"`        | storage driver (memory/redis)                                                                                             |          `memory`          |       `STORAGE_DRIVER`       |
| `--session-ttl="…"`           | session TTL (time-to-live, lifetime)                                                                                      |         `168h0m0s`         |        `SESSION_TTL`         |
| `--max-requests="…"`          | maximal number of requests to store in the storage (zero means unlimited)                                                 |           `128`            |        `MAX_REQUESTS`        |
| `--max-request-body-size="…"` | maximal webhook request body size (in bytes), zero means unlimited                                                        |            `0`             |   `MAX_REQUEST_BODY_SIZE`    |
| `--auto-create-sessions`      | automatically create sessions for incoming requests                                                                       |          `false`           |    `AUTO_CREATE_SESSIONS`    |
| `--pubsub-driver="…"`         | pub/sub driver (memory/redis)                                                                                             |          `memory`          |       `PUBSUB_DRIVER`        |
| `--redis-dsn="…"`             | redis-like (redis, keydb) server DSN (e.g. redis://user:pwd@127.0.0.1:6379/0 or unix://user:pwd@/path/to/redis.sock?db=0) | `redis://127.0.0.1:6379/0` |         `REDIS_DSN`          |
| `--shutdown-timeout="…"`      | maximum duration for graceful shutdown                                                                                    |           `15s`            |      `SHUTDOWN_TIMEOUT`      |
| `--use-live-frontend`         | use frontend from the local directory instead of the embedded one (useful for development)                                |          `false`           |            *none*            |

### `start healthcheck` subcommand (aliases: `hc`, `health`, `check`)

Health checker for the HTTP(S) servers. Use case - docker healthcheck.

Usage:

```bash
$ app [GLOBAL FLAGS] start healthcheck [COMMAND FLAGS] [ARGUMENTS...]
```

The following flags are supported:

| Name              | Description      | Default value | Environment variables |
|-------------------|------------------|:-------------:|:---------------------:|
| `--http-port="…"` | HTTP server port |    `8080`     |      `HTTP_PORT`      |

<!--/GENERATED:CLI_DOCS-->
