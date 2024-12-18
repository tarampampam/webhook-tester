<p align="center">
  <a href="https://github.com/tarampampam/webhook-tester#readme">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://socialify.git.ci/tarampampam/webhook-tester/image?description=1&font=Raleway&forks=1&issues=1&logo=https%3A%2F%2Fgithub.com%2Fuser-attachments%2Fassets%2Fe2e659dc-7fb1-4ac2-ad3c-883899f5fc38&owner=1&pulls=1&pattern=Solid&stargazers=1&theme=Dark">
      <img align="center" src="https://socialify.git.ci/tarampampam/webhook-tester/image?description=1&font=Raleway&forks=1&issues=1&logo=https%3A%2F%2Fgithub.com%2Fuser-attachments%2Fassets%2Fe2e659dc-7fb1-4ac2-ad3c-883899f5fc38&owner=1&pulls=1&pattern=Solid&stargazers=1&theme=Light">
    </picture>
  </a>
</p>

# WebHook Tester

This application allows you to test and debug webhooks and HTTP requests using unique, randomly generated URLs. You
can customize the response code, `Content-Type` HTTP header, response content, and even set a delay for responses.

Consider it a free and self-hosted alternative to [webhook.site](https://github.com/fredsted/webhook.site),
[requestinspector.com](https://requestinspector.com/), and similar services.

<p align="center">
  <img src="https://github.com/user-attachments/assets/26e56d78-8a10-4883-9052-d18047206fda" alt="screencast" />
</p>

> [!TIP]
> The demo is available at [wh.tarampamp.am](https://wh.tarampamp.am/). Please note that it is quite limited, does
> not persist data, and may be unavailable at times, but feel free to give it a try.

Built with Go for high performance, this application includes a lightweight UI (written in `ReactJS`) that’s compiled
into the binary, so no additional assets are required. WebSocket support provides real-time webhook notifications in
the UI - no need for third-party solutions like `pusher.com`!

### 🔥 Features list

- Standalone operation with in-memory storage/pubsub - no third-party dependencies needed
- Fully customizable response code, headers, and body for webhooks
- Option to expose your locally running instance to the global internet (via tunneling)
- Fast, built-in UI based on `ReactJS`
- Multi-architecture Docker image based on `scratch`
- Runs as an unprivileged user in Docker
- Well-tested, documented source code
- CLI health check sub-command included
- Binary view of recorded requests in UI
- Supports JSON and human-readable logging formats
- Liveness probes (`/healthz` endpoint)
- Customizable webhook responses
- Built-in WebSocket support
- Efficient in memory and CPU usage
- Free, open-source, and scalable

### 🗃 Storage

The app supports 3 storage drivers: **memory**, **Redis** and **fs** (configured with the `--storage-driver` flag).

- **Memory** driver: Ideal for local debugging when persistent storage isn’t needed, as recorded requests are cleared
  upon app shutdown
- **Redis** driver: Retains data across app restarts, suitable for environments where data persistence is required.
  Redis is also necessary when running multiple instances behind a load balancer
- **FS** driver: Keep all the data in the local filesystem, useful when you need to store data between app restarts

### 📢 Pub/Sub

For WebSocket notifications, two drivers are supported for the pub/sub system: **memory** and **Redis** (configured
with the `--pubsub-driver` flag).

When running multiple instances of the app, the Redis driver is required.

### 🚀 Tunneling

Capture webhook requests from the global internet using the `ngrok` tunnel driver. Enable it by setting the
`--tunnel-driver=ngrok` flag and providing your `ngrok` authentication token with `--ngrok-auth-token`. Once enabled,
the app automatically creates the tunnel for you – no need to install or run `ngrok` manually (even using docker).

With this public URL, you can test your webhooks from external services like GitHub, GitLab, Bitbucket, and more.
You'll never miss a request!

## 🧩 Installation

Download the latest binary for your architecture from the [releases page][link_releases]. For example, to install
on an **amd64** system (e.g., Debian, Ubuntu):

[link_releases]:https://github.com/tarampampam/webhook-tester/releases

```shell
curl -SsL -o ./webhook-tester https://github.com/tarampampam/webhook-tester/releases/latest/download/webhook-tester-linux-amd64
chmod +x ./webhook-tester
./webhook-tester start
```

> [!TIP]
> Each release includes binaries for **linux**, **darwin** (macOS) and **windows** (`amd64` and `arm64` architectures).
> You can download the binary for your system from the [releases page][link_releases] (section `Assets`). And - yes,
> all what you need is just download and run single binary file.

Alternatively, you can use the Docker image:

| Registry                               | Image                                |
|----------------------------------------|--------------------------------------|
| [GitHub Container Registry][link_ghcr] | `ghcr.io/tarampampam/webhook-tester` |
| [Docker Hub][link_docker_hub] (mirror) | `tarampampam/webhook-tester`         |

> [!NOTE]
> It’s recommended to avoid using the `latest` tag, as **major** upgrades may include breaking changes.
> Instead, use specific tags in `X.Y.Z` format for version consistency.

## ⚙ Usage

The easiest way to run the app is by using the Docker image:

```shell
docker run --rm -t -p "8080:8080/tcp" ghcr.io/tarampampam/webhook-tester:2
```

> [!NOTE]
> This command starts the app with the default configuration on port `8080` (the first port in the `-p` argument is
> the host port, and the second is the application port inside the container).

Next, open your browser at [`localhost:8080`](http://localhost:8080) to begin testing your webhooks. To stop the app, press `Ctrl+C` in
the terminal where it's running.

For custom configuration options, refer to the CLI help below or execute the app with the `--help` flag.

[link_ghcr]:https://github.com/users/tarampampam/packages/container/package/webhook-tester
[link_docker_hub]:https://hub.docker.com/r/tarampampam/webhook-tester/

<!--GENERATED:CLI_DOCS-->
<!-- Documentation inside this block generated by github.com/urfave/cli-docs/v3; DO NOT EDIT -->
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
| `--port="…"`                  | HTTP server port                                                                                                          |           `8080`           |         `HTTP_PORT`          |
| `--read-timeout="…"`          | maximum duration for reading the entire request, including the body (zero = no timeout)                                   |           `1m0s`           |     `HTTP_READ_TIMEOUT`      |
| `--write-timeout="…"`         | maximum duration before timing out writes of the response (zero = no timeout)                                             |           `1m0s`           |     `HTTP_WRITE_TIMEOUT`     |
| `--idle-timeout="…"`          | maximum amount of time to wait for the next request (keep-alive, zero = no timeout)                                       |           `1m0s`           |     `HTTP_IDLE_TIMEOUT`      |
| `--storage-driver="…"`        | storage driver (memory/redis/fs)                                                                                          |          `memory`          |       `STORAGE_DRIVER`       |
| `--session-ttl="…"`           | session TTL (time-to-live, lifetime)                                                                                      |         `168h0m0s`         |        `SESSION_TTL`         |
| `--max-requests="…"`          | maximal number of requests to store in the storage (zero means unlimited)                                                 |           `128`            |        `MAX_REQUESTS`        |
| `--fs-storage-dir="…"`        | path to the directory for local fs storage (directory must exist)                                                         |                            |       `FS_STORAGE_DIR`       |
| `--max-request-body-size="…"` | maximal webhook request body size (in bytes), zero means unlimited                                                        |            `0`             |   `MAX_REQUEST_BODY_SIZE`    |
| `--auto-create-sessions`      | automatically create sessions for incoming requests                                                                       |          `false`           |    `AUTO_CREATE_SESSIONS`    |
| `--pubsub-driver="…"`         | pub/sub driver (memory/redis)                                                                                             |          `memory`          |       `PUBSUB_DRIVER`        |
| `--tunnel-driver="…"`         | tunnel driver to expose your locally running app to the internet (ngrok, empty to disable)                                |                            |       `TUNNEL_DRIVER`        |
| `--ngrok-auth-token="…"`      | ngrok authentication token (required for ngrok tunnel; create a new one at https://dashboard.ngrok.com/authtokens/new)    |                            |      `NGROK_AUTHTOKEN`       |
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

| Name         | Description      | Default value | Environment variables |
|--------------|------------------|:-------------:|:---------------------:|
| `--port="…"` | HTTP server port |    `8080`     |      `HTTP_PORT`      |

<!--/GENERATED:CLI_DOCS-->

## License

This is open-sourced software licensed under the [MIT License][link_license].

[link_license]:https://github.com/tarampampam/webhook-tester/blob/master/LICENSE
