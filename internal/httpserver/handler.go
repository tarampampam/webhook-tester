package httpserver

import (
	"net/http"
	"strings"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/frontend"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/webhook"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/web"
)

// NewHandler creates a new HTTP handler that serves all server endpoints, including the OpenAPI endpoints and
// the SPA frontend.
func NewHandler(
	log *logger.Logger,
	s storage.Storage,
	ps pubsub.PubSub,
	sessionTTL time.Duration,
	appSettings AppSettings,
	autoCreateSessions bool,
	maxRequestBodySize uint,
	readyChecker checker,
	latestVersionGetter latestVersionProvider,
	useLiveFrontend bool,
) http.Handler {
	mux := http.NewServeMux()
	spa := frontend.New(web.Dist(useLiveFrontend))
	api := NewOpenAPI(log, s, ps, sessionTTL, appSettings, readyChecker, latestVersionGetter)

	// do you get it? it's a webhook handler, so "huk" :D
	huk := webhook.New(log.Named("webhook"), s, ps, autoCreateSessions, sessionTTL, maxRequestBodySize)

	handler := openapi.HandlerWithOptions(api, openapi.StdHTTPServerOptions{
		ErrorHandlerFunc: api.HandleInternalError, // set error handler for internal server errors
		BaseRouter:       mux,
	})

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// custom logic for handling 404 errors for API routes
		if strings.HasPrefix(strings.TrimLeft(r.URL.Path, "/"), "api") {
			api.HandleNotFoundError(w, r)

			return
		}

		// if the request is a webhook request, handle it with the webhook handler
		if webhook.ShouldBeCaptured(r) {
			huk.Handle(w, r)

			return
		}

		// otherwise, serve the SPA frontend
		spa.ServeHTTP(w, r)
	}))

	return middleware.Apply(handler,
		middleware.NewOriginalURL(),
		middleware.PreCleanPath,
		middleware.NewInjectLog(log),
		middleware.NewAccessLog(logger.InfoLevel, func() func(*http.Request) bool {
			// the following routes will NOT be logged
			skipMap := map[string]struct{}{
				strings.ToLower(openapi.RouteLivenessProbe):  {},
				strings.ToLower(openapi.RouteReadinessProbe): {},
				"/apple-touch-icon.png":                      {},
				"/site.webmanifest":                          {},
				"/favicon.ico":                               {},
				"/favicon.svg":                               {},
				"/robots.txt":                                {},
			}

			return func(r *http.Request) bool {
				_, skip := skipMap[strings.ToLower(r.URL.Path)]

				return skip
			}
		}()),
		middleware.NewCacheControl(nil),
	)
}
