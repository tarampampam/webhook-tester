package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"

	"gh.tarampamp.am/webhook-tester/v3/cmd/webhook-tester/app/loader"
	"gh.tarampamp.am/webhook-tester/v3/internal/appmeta"
	"gh.tarampamp.am/webhook-tester/v3/internal/cli"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/cert"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/http2https"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/tunnel"
)

//go:generate go run ./generate/readme.go -out ../../../README.md

// App represents the CLI application with its command and options.
type App struct {
	cmd cli.Command

	opt struct {
		http struct {
			addr              string
			port              uint
			tlsCert           *tls.Certificate
			readTimeout       time.Duration
			idleTimeout       time.Duration
			readHeaderTimeout time.Duration
			shutdownTimeout   time.Duration
		}
		redis   struct{ dsn string }
		pubSub  struct{ driver pubSubDriverName }
		storage struct {
			driver      storageDriverName
			sessionTTL  time.Duration
			maxRequests uint
			fsDir       string
		}
		tunnel struct {
			driver tunnelDriverName
			url    string
			ngrok  struct {
				authToken string
			}
		}
		maxRequestBodySize uint
		autoCreateSessions bool
		publicURLRoot      string // public URL root override
		useLiveFrontend    bool
		sessionsFile       string // path to a JSON file with pre-configured sessions to create on startup
	}
}

// NewApp initializes a new CLI application instance.
func NewApp(name string) *App { //nolint:funlen
	app := App{
		cmd: cli.Command{
			Name:        name,
			Description: "Start the HTTP server to receive and display incoming webhooks",
			Version:     appmeta.Version(),
		},
	}

	app.opt.http.addr = "0.0.0.0" // bind to all interfaces by default
	app.opt.http.port = 8080
	app.opt.http.readHeaderTimeout = 5 * time.Second //nolint:mnd
	app.opt.http.idleTimeout = 60 * time.Second      //nolint:mnd
	app.opt.http.shutdownTimeout = 5 * time.Second   //nolint:mnd
	app.opt.pubSub.driver = pubSubDriverMemory
	app.opt.storage.driver = storageDriverMemory
	app.opt.storage.maxRequests = 128
	app.opt.storage.sessionTTL = time.Hour * 24 * 7 //nolint:mnd // 7 days

	var (
		logLevelFlag              = newLogLevelFlag()
		logFormatFlag             = newLogFormatFlag()
		httpAddrFlag              = newHTTPAddrFlag(app.opt.http.addr)
		httpPortFlag              = newHTTPPortFlag(app.opt.http.port)
		httpReadHeaderTimeoutFlag = newHTTPReadHeaderTimeoutFlag(app.opt.http.readHeaderTimeout)
		httpReadTimeoutFlag       = newHTTPReadTimeoutFlag(app.opt.http.readTimeout)
		httpIdleTimeoutFlag       = newHTTPIdleTimeoutFlag(app.opt.http.idleTimeout)
		httpShutdownTimeoutFlag   = newHTTPShutdownTimeoutFlag(app.opt.http.shutdownTimeout)
		tlsKeyFileFlag            = newTLSKeyFileFlag()
		tlsCertFileFlag           = newTLSCertFileFlag()
		selfSigningKeyFlag        = newSelfSignedTLSFlag()
		redisServerDsnFlag        = newRedisServerDsnFlag(app.opt.redis.dsn)
		pubSubDriverFlag          = newPubSubDriverFlag(app.opt.pubSub.driver)
		storageDriverFlag         = newStorageDriverFlag(app.opt.storage.driver)
		maxRequestsFlag           = newMaxRequestsFlag(app.opt.storage.maxRequests)
		sessionTTLFlag            = newSessionTTLFlag(app.opt.storage.sessionTTL)
		fsStorageDirFlag          = newFSStorageDirFlag(app.opt.storage.fsDir)
		maxRequestBodySizeFlag    = newMaxRequestBodySizeFlag(app.opt.maxRequestBodySize)
		autoCreateSessionsFlag    = newAutoCreateSessionsFlag()
		publicURLRootFlag         = newPublicURLRootFlag(app.opt.publicURLRoot)
		tunnelDriverFlag          = newTunnelDriverFlag(app.opt.tunnel.driver)
		tunnelURLFlag             = newTunnelURLFlag(app.opt.tunnel.url)
		tunnelNgrokAuthTokenFlag  = newTunnelNgrokAuthTokenFlag()
		useLiveEndpointFlag       = newUseLiveEndpointFlag()
		sessionsFileFlag          = newSessionsFileFlag()
	)

	app.cmd.Flags = []cli.Flagger{
		&logLevelFlag,
		&logFormatFlag,
		&httpAddrFlag,
		&httpPortFlag,
		&httpReadHeaderTimeoutFlag,
		&httpReadTimeoutFlag,
		&httpIdleTimeoutFlag,
		&httpShutdownTimeoutFlag,
		&tlsKeyFileFlag,
		&tlsCertFileFlag,
		&selfSigningKeyFlag,
		&redisServerDsnFlag,
		&pubSubDriverFlag,
		&storageDriverFlag,
		&maxRequestsFlag,
		&sessionTTLFlag,
		&fsStorageDirFlag,
		&maxRequestBodySizeFlag,
		&autoCreateSessionsFlag,
		&publicURLRootFlag,
		&tunnelDriverFlag,
		&tunnelURLFlag,
		&tunnelNgrokAuthTokenFlag,
		&useLiveEndpointFlag,
		&sessionsFileFlag,
	}

	app.cmd.Action = func(ctx context.Context, _ *cli.Command, _ []string) error {
		var (
			logLevel, _  = logger.ParseLevel(*logLevelFlag.Value)   //nolint:errcheck // because the flag validates itself
			logFormat, _ = logger.ParseFormat(*logFormatFlag.Value) //nolint:errcheck // format flag validates itself
		)

		log, logErr := logger.New(logLevel, logFormat)
		if logErr != nil {
			return logErr
		}

		// put logger into context as early as possible to be able to use it in the rest of the application
		ctx = logger.With(ctx, log)

		setIfFlagIsSet(&app.opt.http.addr, httpAddrFlag)
		setIfFlagIsSet(&app.opt.http.port, httpPortFlag)
		setIfFlagIsSet(&app.opt.http.readHeaderTimeout, httpReadHeaderTimeoutFlag)
		setIfFlagIsSet(&app.opt.http.readTimeout, httpReadTimeoutFlag)
		setIfFlagIsSet(&app.opt.http.idleTimeout, httpIdleTimeoutFlag)
		setIfFlagIsSet(&app.opt.http.shutdownTimeout, httpShutdownTimeoutFlag)

		// load TLS certificate and key from files if both flags are set
		if tlsKeyFileFlag.IsSet() && tlsCertFileFlag.IsSet() {
			tlsCert, certErr := tls.LoadX509KeyPair(*tlsCertFileFlag.Value, *tlsKeyFileFlag.Value)
			if certErr != nil {
				return fmt.Errorf("load TLS cert and key: %w", certErr)
			}

			log.Info("Loaded TLS certificate and key from files",
				logger.String("certFile", *tlsCertFileFlag.Value),
				logger.String("keyFile", *tlsKeyFileFlag.Value),
			)

			app.opt.http.tlsCert = &tlsCert
		} else if selfSigningKeyFlag.IsSet() { // otherwise, if self-signed TLS is requested, generate it
			tlsCert, certErr := cert.SelfSignedTLS(os.TempDir())
			if certErr != nil {
				return fmt.Errorf("generate self-signed TLS cert: %w", certErr)
			}

			log.Info("Self-signed TLS certificate generated", logger.String("subject", tlsCert.Leaf.Subject.CommonName))

			app.opt.http.tlsCert = tlsCert
		}

		// set Redis DSN if the flag is set (it can be left empty, which means no Redis)
		setIfFlagIsSet(&app.opt.redis.dsn, redisServerDsnFlag)

		// set pub/sub driver
		if pubSubDriverFlag.Value == nil {
			return errors.New("pub/sub driver flag is not set")
		}

		app.opt.pubSub.driver = pubSubDriverName(*pubSubDriverFlag.Value) // validated by the flag, so it must be correct

		// set storage driver
		if storageDriverFlag.Value == nil {
			return errors.New("storage driver flag is not set")
		}

		app.opt.storage.driver = storageDriverName(*storageDriverFlag.Value) // is also validated by the flag

		setIfFlagIsSet(&app.opt.storage.maxRequests, maxRequestsFlag)
		setIfFlagIsSet(&app.opt.storage.sessionTTL, sessionTTLFlag)
		setIfFlagIsSet(&app.opt.storage.fsDir, fsStorageDirFlag)
		setIfFlagIsSet(&app.opt.maxRequestBodySize, maxRequestBodySizeFlag)
		setIfFlagIsSet(&app.opt.autoCreateSessions, autoCreateSessionsFlag)
		setIfFlagIsSet(&app.opt.publicURLRoot, publicURLRootFlag)

		// set tunnel driver
		if tunnelDriverFlag.Value != nil {
			app.opt.tunnel.driver = tunnelDriverName(*tunnelDriverFlag.Value) // is also validated by the flag
		}

		setIfFlagIsSet(&app.opt.tunnel.url, tunnelURLFlag)
		setIfFlagIsSet(&app.opt.tunnel.ngrok.authToken, tunnelNgrokAuthTokenFlag)

		// if ngrok auth token is set but tunnel driver is not specified, assume ngrok
		if app.opt.tunnel.ngrok.authToken != "" && app.opt.tunnel.driver == "" {
			app.opt.tunnel.driver = tunnelDriverNgrok
		}

		setIfFlagIsSet(&app.opt.useLiveFrontend, useLiveEndpointFlag)
		setIfFlagIsSet(&app.opt.sessionsFile, sessionsFileFlag)

		// and run the application (finally!)
		if err := app.run(ctx, log); err != nil {
			log.Error("Application error", logger.Error(err))

			return errors.New("http server failed")
		}

		return nil
	}

	return &app
}

// Help returns the help message.
func (a *App) Help() string { return a.cmd.Help() }

// setIfFlagIsSet copies source's value into target only if the flag was explicitly provided by the user, not just
// defaulted. This matters because Flag.Value is always non-nil (set to default) after parsing, so IsSet is the only
// reliable way to know whether the user actually supplied the value.
func setIfFlagIsSet[T cli.FlagType](target *T, source cli.Flag[T]) {
	if target == nil || source.Value == nil || !source.IsSet() {
		return
	}

	*target = *source.Value
}

// Run starts the CLI command execution.
func (a *App) Run(ctx context.Context, args []string) error { return a.cmd.Run(ctx, args) }

func (a *App) run(ctx context.Context, log *logger.Logger) error { //nolint:gocognit,funlen
	// establish Redis connection if Redis is selected as a driver for either pub/sub or storage (or both)
	var redisClient redis.UniversalClient // may be nil - that's totally fine, it just means that Redis is not needed

	if a.opt.pubSub.driver == pubSubDriverRedis || a.opt.storage.driver == storageDriverRedis {
		redisLog := log.Named("redis")

		client, redisErr := a.newRedisClient(redisLog)
		if redisErr != nil {
			return redisErr
		}

		defer func() {
			if err := client.Close(); err != nil {
				redisLog.Error("Failed to close Redis client", logger.Error(err))
			}
		}()

		if pingErr := client.Ping(ctx).Err(); pingErr != nil {
			return fmt.Errorf("connect to Redis: %w", pingErr)
		}

		redisClient = client
	}

	// open root directory for FS storage if it's selected as the driver
	var rootFs *os.Root // may be nil - that's also fine, it just means that FS storage is not needed

	if a.opt.storage.driver == storageDriverFS {
		if a.opt.storage.fsDir == "" {
			return errors.New("FS storage driver requires a path to the directory where all data will be stored")
		}

		root, rErr := os.OpenRoot(a.opt.storage.fsDir)
		if rErr != nil {
			return fmt.Errorf("open root directory for FS storage: %w", rErr)
		}

		defer func() {
			if err := root.Close(); err != nil {
				log.Error("Failed to close root directory for FS storage", logger.Error(err))
			}
		}()

		rootFs = root
	}

	// initialize storage based on the selected driver
	strg, strgErr := a.newStorage(ctx, redisClient, rootFs)
	if strgErr != nil {
		return fmt.Errorf("initialize storage: %w", strgErr)
	}

	if v, ok := strg.(io.Closer); ok { // close storage automatically if it implements io.Closer
		defer func() {
			if err := v.Close(); err != nil {
				log.Error("Failed to close storage", logger.Error(err))
			}
		}()
	}

	// https://github.com/tarampampam/webhook-tester/issues/771
	if err := a.loadSessions(ctx, log.Named("bootstrap"), strg); err != nil {
		return fmt.Errorf("load sessions: %w", err)
	}

	// initialize pub/sub based on the selected driver
	ps, psErr := a.newPubSub(redisClient)
	if psErr != nil {
		return fmt.Errorf("initialize pub/sub: %w", psErr)
	}

	if v, ok := ps.(io.Closer); ok { // close pub/sub automatically if it implements io.Closer
		defer func() {
			if err := v.Close(); err != nil {
				log.Error("Failed to close pub/sub", logger.Error(err))
			}
		}()
	}

	// we use atomic string here because the tunnel will update the public URL dynamically after it starts
	var tnlPublicUrl atomic.Pointer[string]

	// initialize tunnel based on the selected driver (if any)
	tnl, tnlErr := a.newTunnel(log.Named("tunnel"))
	if tnlErr != nil {
		return fmt.Errorf("initialize tunnel: %w", tnlErr)
	} else if tnl != nil {
		if v, ok := tnl.(io.Closer); ok { // close tunnel automatically if it implements io.Closer
			defer func() {
				if err := v.Close(); err != nil {
					log.Error("Failed to close tunnel", logger.Error(err))
				}
			}()
		}
	}

	// define a health check function that will be used by the HTTP server to report its health status
	healthyChecker := func(ctx context.Context) error {
		if redisClient != nil {
			if err := redisClient.Ping(ctx).Err(); err != nil {
				return err
			}
		}

		return nil
	}

	// initialize HTTP server
	srv := a.newHTTPServer(log.Named("http"), strg, ps, healthyChecker, &tnlPublicUrl)

	log.Info("Opening TCP port",
		logger.String("addr", a.opt.http.addr),
		logger.Uint64("port", uint64(a.opt.http.port)),
	)

	// open TCP port for the HTTP(s) server to listen on
	ln, lnErr := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort(
		a.opt.http.addr,
		strconv.Itoa(int(a.opt.http.port)),
	))
	if lnErr != nil {
		return fmt.Errorf("listen http: %w", lnErr)
	}
	defer func() { _ = ln.Close() }()

	// when TLS is enabled, wrap the listener so plain HTTP requests are redirected to HTTPS automatically;
	// the actual TLS handshake is still performed by server.go via tls.NewListener inside httpserver.Serve
	if a.opt.http.tlsCert != nil {
		ln = http2https.NewListener(ln)

		log.Info("HTTP requests will be redirected to HTTPS automatically")
	}

	// capture current time to be able to log uptime on shutdown
	now := time.Now()

	defer func() { log.Info("HTTP server stopped", logger.Duration("uptime", time.Since(now))) }()

	log.Info("HTTP server started", logger.String("addr", ln.Addr().String()))

	// start the tunnel in a separate goroutine since it may take some time to establish the connection, and we don't
	// want to block the main server loop
	if tnl != nil {
		go func() {
			var port uint16
			if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok {
				port = uint16(tcpAddr.Port) //nolint:gosec // it's safe, trust me
			} else {
				log.Error("Failed to get local port for tunnel", logger.String("addr", ln.Addr().String()))

				return
			}

			pubUrl, err := tnl.Expose(ctx, port)
			if err != nil {
				log.Error("Failed to open tunnel", logger.Error(err))

				return
			}

			tnlPublicUrl.Store(new(pubUrl))

			log.Info("Tunnel opened", logger.String("public_url", pubUrl))
		}()
	}

	return srv.Serve(ctx, ln)
}

// newRedisClient creates a new Redis client based on initialized options and returns it along with a closer
// function to clean up resources on shutdown.
func (a *App) newRedisClient(log *logger.Logger) (redis.UniversalClient, error) {
	opt, pErr := redis.ParseURL(a.opt.redis.dsn)
	if pErr != nil {
		return nil, fmt.Errorf("parse Redis DSN: %w", pErr)
	}

	// disable maintenance notifications (https://github.com/tarampampam/webhook-tester/issues/713)
	if opt.MaintNotificationsConfig == nil {
		opt.MaintNotificationsConfig = &maintnotifications.Config{}
	}

	opt.MaintNotificationsConfig.Mode = maintnotifications.ModeDisabled

	client := redis.NewClient(opt)

	redis.SetLogger(logger.NewRedisBridge(log, logger.WarnLevel)) // set global Redis logger to our

	return client, nil
}

// newStorage creates a new storage instance based on the selected driver and initialized options.
func (a *App) newStorage(ctx context.Context, r redis.UniversalClient, rootFs *os.Root) (storage.Storage, error) {
	switch a.opt.storage.driver {
	case storageDriverMemory:
		s := storage.NewMemory(ctx, a.opt.storage.maxRequests)

		return s, nil
	case storageDriverRedis:
		if r == nil {
			return nil, errors.New("redis storage requires Redis client to be initialized")
		}

		s := storage.NewRedis(r, a.opt.storage.maxRequests)

		return s, nil
	case storageDriverFS:
		if rootFs == nil {
			return nil, errors.New("FS storage requires root directory to be opened")
		}

		s := storage.NewFS(ctx, rootFs, a.opt.storage.maxRequests)

		// reindex existing data once at startup
		if err := s.Reindex(ctx); err != nil {
			_ = s.Close()

			return nil, err
		}

		return s, nil
	default:
		return nil, fmt.Errorf("unsupported storage driver: %q", a.opt.storage.driver)
	}
}

// loadSessions loads predefined sessions from a file and creates them in the storage.
func (a *App) loadSessions(ctx context.Context, log *logger.Logger, s storage.SessionStorage) error {
	if a.opt.sessionsFile == "" {
		return nil
	}

	list, sErr := loader.LoadSessions(a.opt.sessionsFile)
	if sErr != nil {
		return fmt.Errorf("load sessions from file: %w", sErr)
	}

	for id, resp := range list {
		if _, err := s.NewSession(ctx, id, resp, a.opt.storage.sessionTTL); err != nil {
			if errors.Is(err, storage.ErrSessionAlreadyExists) {
				log.Debug("Session already exists, skipping", logger.String("session_id", id))

				continue
			}

			return fmt.Errorf("create session %q: %w", id, err)
		}
	}

	return nil
}

// newPubSub creates a new pub/sub instance based on the selected driver and initialized options.
func (a *App) newPubSub(r redis.UniversalClient) (pubsub.PubSub, error) {
	switch a.opt.pubSub.driver {
	case pubSubDriverMemory:
		ps := pubsub.NewMemory()

		return ps, nil
	case pubSubDriverRedis:
		if r == nil {
			return nil, errors.New("redis pub/sub requires Redis client to be initialized")
		}

		ps := pubsub.NewRedis(r)

		return ps, nil
	default:
		return nil, fmt.Errorf("unsupported Pub/Sub driver: %q", a.opt.pubSub.driver)
	}
}

// newTunnel creates a new tunnel instance, it may return nil if no tunnel driver is selected, which is also fine.
func (a *App) newTunnel(log *logger.Logger) (tunnel.Tunneler, error) {
	if a.opt.tunnel.driver == "" {
		return nil, nil //nolint:nilnil
	}

	switch a.opt.tunnel.driver {
	case tunnelDriverNgrok:
		if a.opt.tunnel.ngrok.authToken == "" {
			return nil, errors.New("ngrok tunnel driver requires auth token to be set")
		}

		opts := []tunnel.NgrokOption{tunnel.WithNgrokLogger(log.Named("ngrok"))}

		if v := a.opt.tunnel.url; v != "" {
			opts = append(opts, tunnel.WithNgrokURL(v))
		}

		if a.opt.http.tlsCert != nil {
			opts = append(opts, tunnel.WithNgrokTLSUpstream(&tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // connecting to our own local upstream with a self-signed cert
			}))
		}

		return tunnel.NewNgrok(a.opt.tunnel.ngrok.authToken, opts...), nil
	default:
		return nil, fmt.Errorf("unsupported tunnel driver: %q", a.opt.tunnel.driver)
	}
}

func (a *App) newHTTPServer(
	log *logger.Logger,
	strg storage.Storage,
	ps pubsub.PubSub,
	healthyChecker func(context.Context) error,
	tnlPublicUrl *atomic.Pointer[string],
) *httpserver.Server {
	opts := []httpserver.Option{
		httpserver.WithErrorLog(logger.NewStdLog(log, logger.ErrorLevel)),
		httpserver.WithReadHeaderTimeout(a.opt.http.readHeaderTimeout),
		httpserver.WithReadTimeout(a.opt.http.readTimeout),
		httpserver.WithIdleTimeout(a.opt.http.idleTimeout),
		httpserver.WithShutdownTimeout(a.opt.http.shutdownTimeout),
	}

	// switch HTTP server to HTTPS mode if TLS certificate is provided
	if a.opt.http.tlsCert != nil {
		opts = append(opts, httpserver.WithTLSConfig(&tls.Config{
			Certificates: []tls.Certificate{*a.opt.http.tlsCert},
		}))

		log.Info("HTTPS enabled with TLS certificate",
			logger.String("subject", a.opt.http.tlsCert.Leaf.Subject.CommonName),
			logger.Strings("domain_names", a.opt.http.tlsCert.Leaf.DNSNames...),
			logger.Time("not_before", a.opt.http.tlsCert.Leaf.NotBefore),
			logger.Time("not_after", a.opt.http.tlsCert.Leaf.NotAfter),
		)
	}

	settings := httpserver.AppSettings{
		MaxRequestBodySize: uint32(a.opt.maxRequestBodySize),  //nolint:gosec // validated by the flag
		MaxRequests:        uint16(a.opt.storage.maxRequests), //nolint:gosec // validated by the flag
		PublicUrlRoot:      a.opt.publicURLRoot,
		TunnelUrl:          tnlPublicUrl,
	}

	return httpserver.New(
		httpserver.NewHandler(
			log,
			strg,
			ps,
			a.opt.storage.sessionTTL,
			settings,
			a.opt.autoCreateSessions,
			a.opt.maxRequestBodySize,
			healthyChecker,
			func(ctx context.Context) (string, error) { return appmeta.Latest(ctx) },
			a.opt.useLiveFrontend,
		),
		opts...,
	)
}
