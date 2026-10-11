<!--
🤖 AI CODING AGENT - IF YOU ARE READING THIS, DO NOT IGNORE THIS BLOCK.

Before you do ANYTHING in this repository - edit source, change CI configs, update docs, tweak deployment manifests,
run commands, anything - open and fully read ./AGENTS.md in the repo root. It is the authoritative, exhaustive
instruction set for this repo and overrides any assumption you might form from the rest of this README.
-->

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
> The demo is available at [wh.tarampamp.am](https://wh.tarampamp.am/). Please note that it is quite limited,
> does not persist data, and may be unavailable sometimes, but feel free to try it.

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

## ⁉ FAQ

**Can I have pre-defined (static) webhook URLs (sessions) to ensure that the sent request will be captured even
without data persistence?**

Yes, simply use the `--auto-create-sessions` flag or set the `AUTO_CREATE_SESSIONS=true` environment variable. In
`v1`, you needed to define sessions during app startup to enable this functionality. However, since `v2`, all you
need to do is enable this feature. It works quite simply - if the incoming request contains a UUID-formatted prefix
(e.g., `http://app/11111111-2222-3333-4444-555555555555/...`), a session for this request will be created
automatically. All that's left for you to do is open the session in the UI
(`http://app/s/11111111-2222-3333-4444-555555555555`).

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

To install it on Kubernetes (K8s), please use the Helm chart from [ArtifactHUB][artifact-hub].

[artifact-hub]:https://artifacthub.io/packages/helm/webhook-tester/webhook-tester

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

# 💻 Command line interface

<!--GENERATED:WEBHOOK_TESTER_CLI-->
```
Description:
   Start the HTTP server to receive and display incoming webhooks

Usage:
   webhook-tester

Version:
   0.0.0@undefined

Options:
   --log-level="…"                 Logging level (debug/info/warn/error) (default: info) [$LOG_LEVEL]
   --log-format="…"                Logging format (console/json) (default: console) [$LOG_FORMAT]
   --addr="…", --listen="…"        HTTP server address to listen on (IPv4 or IPv6) (default: 0.0.0.0) [$HTTP_ADDR, $LISTEN_ADDR, $ADDR]
   --port="…"                      HTTP server TCP port number (default: 8080) [$HTTP_PORT, $LISTEN_PORT, $PORT]
   --http-read-header-timeout="…"  Maximum time allowed to read request headers; set 0 to disable (not recommended - makes server vulnerable to Slowloris) (default: 5s) [$HTTP_READ_HEADER_TIMEOUT]
   --http-read-timeout="…"         Maximum duration for reading the entire request including the body; set 0 to disable [$HTTP_READ_TIMEOUT]
   --http-idle-timeout="…"         Maximum time to wait for the next request when keep-alives are enabled; set 0 to fall back to read timeout (default: 1m0s) [$HTTP_IDLE_TIMEOUT]
   --http-shutdown-timeout="…"     Maximum time to wait for in-flight requests to complete on graceful shutdown; set 0 for immediate close (default: 5s) [$HTTP_SHUTDOWN_TIMEOUT]
   --https-key-file="…"            Path to the TLS private key file for HTTPS [$TLS_KEY_FILE, $HTTPS_KEY_FILE]
   --https-cert-file="…"           Path to the TLS certificate file for HTTPS [$TLS_CERT_FILE, $HTTPS_CERT_FILE]
   --self-signed-tls               Generate and use a self-signed TLS certificate for HTTPS (ignored if TLS cert/key files are provided) [$SELF_SIGNED_CERT]
   --redis-dsn="…"                 Redis-like (redis/reydb) server DSN (e.g. redis://user:pass@host:port/db or unix://user:pass@/var/run/redis/redis.sock?db=0) [$REDIS_DSN]
   --pubsub-driver="…"             Pub/Sub driver to use for broadcasting events via WebSocket (supported: memory/redis) (default: memory) [$PUBSUB_DRIVER]
   --storage-driver="…"            Storage driver to use for storing received requests (supported: memory/redis/fs) (default: memory) [$STORAGE_DRIVER]
   --max-requests="…"              Maximum number of requests to store in memory (set 0 for unlimited) (default: 128) [$MAX_REQUESTS]
   --session-ttl="…"               Time-to-live for sessions (e.g. 500ms, 145s, 1h30m; set 0 for no expiration) (default: 168h0m0s) [$SESSION_TTL]
   --fs-storage-dir="…"            Directory path for storing received requests when using filesystem storage driver [$FS_STORAGE_DIR]
   --max-request-body-size="…"     Maximum size of the request body in bytes (set 0 for unlimited) [$MAX_REQUEST_BODY_SIZE]
   --auto-create-sessions          Automatically create new sessions for incoming requests [$AUTO_CREATE_SESSION]
   --public-url-root="…"           Public URL root override for webhook URLs (e.g., http://webhook-tester.k8s.internal); if not set, the URL shown in the UI is based on the browser's location [$PUBLIC_URL_ROOT]
   --tunnel-driver="…"             Tunnel driver to use for exposing the server to the public internet (supported: ngrok) [$TUNNEL_DRIVER]
   --tunnel-url="…"                Public URL to use for the tunnel (for ngrok, register it first at https://dashboard.ngrok.com/domains) [$TUNNEL_URL]
   --ngrok-auth-token="…"          Ngrok auth token for tunnel authentication (create a new one at https://dashboard.ngrok.com/authtokens/new) [$NGROK_AUTHTOKEN]
   --use-live-frontend             Serve the frontend assets from the filesystem instead of embedded ones (useful for development)
   --sessions-file="…"             Path to a JSON file with pre-configured sessions to create on startup (existing sessions are skipped) [$SESSIONS_FILE]
   --help, -h                      Show help
   --version, -v                   Print the version
```
<!--/GENERATED:WEBHOOK_TESTER_CLI-->

## 🧠 A note on AI-assisted development

AI tools are great assistants - they can autocomplete, review, summarize, and help you move faster. But they’re not a
substitute for understanding what's going on. If you're using AI to contribute here, please make sure you actually
read, understand, and stand behind the changes you’re proposing.

I personally write my code myself, and I encourage others to do the same. Not because AI is "bad", but because blindly
trusting generated code tends to produce... let's say creative results.

And honestly, I'm still waiting for the day "AI-free software" becomes a trend - like organic food, but for code 😄
Until then: trust, but verify.

## 🤖 AI Agent Instructions

See [AGENTS.md](AGENTS.md) for detailed guidelines for AI agents working with this repository.

## License

This is open-sourced software licensed under the [MIT License][link_license].

[link_license]:https://github.com/tarampampam/webhook-tester/blob/master/LICENSE
