package http

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"
	"webhook-tester/broadcast"
	"webhook-tester/settings"
	"webhook-tester/storage"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type (
	ServerSettings struct {
		Address                   string // TCP address to listen on
		WriteTimeout              time.Duration
		ReadTimeout               time.Duration
		PublicAssetsDirectoryPath string
		KeepAliveEnabled          bool
	}

	Server struct {
		settings    *ServerSettings
		appSettings *settings.AppSettings
		Server      *http.Server
		Router      *mux.Router
		storage     storage.Storage
		broadcaster broadcast.Broadcaster // optional, can be nil
		stdLog      *log.Logger
		errLog      *log.Logger
	}
)

// NewServer creates new server instance.
func NewServer(
	srvSettings *ServerSettings,
	appSettings *settings.AppSettings,
	storage storage.Storage,
	br broadcast.Broadcaster,
) *Server {
	var (
		router     = *mux.NewRouter()
		stdLog     = log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds)
		errLog     = log.New(os.Stderr, "[error] ", log.LstdFlags)
		httpServer = &http.Server{
			Addr:         srvSettings.Address,
			Handler:      handlers.CombinedLoggingHandler(os.Stdout, &router),
			ErrorLog:     errLog,
			WriteTimeout: srvSettings.WriteTimeout,
			ReadTimeout:  srvSettings.ReadTimeout,
		}
	)

	httpServer.SetKeepAlivesEnabled(srvSettings.KeepAliveEnabled)

	return &Server{
		settings:    srvSettings,
		appSettings: appSettings,
		Server:      httpServer,
		Router:      &router,
		storage:     storage,
		broadcaster: br,
		stdLog:      stdLog,
		errLog:      errLog,
	}
}

// Start Server.
func (s *Server) Start() error {
	s.stdLog.Println("Starting Server on " + s.Server.Addr)

	return s.Server.ListenAndServe()
}

// Stop Server.
func (s *Server) Stop(ctx context.Context) error {
	s.stdLog.Println("Stopping Server")

	return s.Server.Shutdown(ctx)
}
