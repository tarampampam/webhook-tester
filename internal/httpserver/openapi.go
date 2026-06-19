package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/appmeta"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/request_delete"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/request_get"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/requests_delete_all"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/requests_list"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/requests_subscribe"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_check_exists"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_create"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_delete"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_get"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/version_latest"
	j "gh.tarampamp.am/webhook-tester/v3/internal/httpserver/json"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

type (
	// sID and rID are type aliases for session and request IDs, respectively, to improve code readability.
	sID = string
	rID = string

	// checker is a function type for performing readiness checks, returning an error if the check fails.
	checker func(context.Context) error
)

// OpenAPI is the server implementation for the OpenAPI specification.
type OpenAPI struct {
	log      *logger.Logger
	handlers struct {
		session struct {
			create func(context.Context, openapi.CreateSessionRequest) (*openapi.CreateSessionResponse, error)
			get    func(context.Context, sID) (*openapi.SessionOptionsResponse, error)
			exists func(context.Context, []sID) (*openapi.CheckSessionExistsResponse, error)
			delete func(context.Context, sID) (*openapi.SuccessfulOperationResponse, error)
		}
		request struct {
			get       func(context.Context, sID, rID) (*openapi.CapturedRequestsResponse, error)
			list      func(context.Context, sID) (*openapi.CapturedRequestsListResponse, error)
			subscribe func(context.Context, http.ResponseWriter, *http.Request, sID) error
			delete    func(context.Context, sID, rID) (*openapi.SuccessfulOperationResponse, error)
			deleteAll func(context.Context, sID) (*openapi.SuccessfulOperationResponse, error)
		}
		version struct {
			latest func(context.Context) (*openapi.VersionResponse, error)
		}
	}
	sessionTTL   time.Duration
	settings     AppSettings
	readyChecker checker
}

// AppSettings is a "holder" struct for application settings that are returned in the /api/settings response.
type AppSettings struct {
	MaxRequestBodySize uint32
	MaxRequests        uint16
	PublicUrlRoot      string
	TunnelUrl          *atomic.Pointer[string]
}

var _ openapi.ServerInterface = (*OpenAPI)(nil) // compile-time interface implementation check

// NewOpenAPI creates a new instance of the OpenAPI server implementation.
func NewOpenAPI(
	log *logger.Logger,
	s storage.Storage,
	ps pubsub.PubSub,
	sessionTTL time.Duration,
	settings AppSettings,
	readyChecker checker,
	latestVersionGetter func(context.Context) (string, error),
) *OpenAPI {
	o := OpenAPI{
		log:          log,
		sessionTTL:   sessionTTL,
		settings:     settings,
		readyChecker: readyChecker,
	}

	o.handlers.session.create = session_create.New(s, sessionTTL).Handle
	o.handlers.session.get = session_get.New(s).Handle
	o.handlers.session.exists = session_check_exists.New(s).Handle
	o.handlers.session.delete = session_delete.New(s).Handle
	o.handlers.request.get = request_get.New(s).Handle
	o.handlers.request.list = requests_list.New(s).Handle
	o.handlers.request.subscribe = requests_subscribe.New(ps).Handle
	o.handlers.request.delete = request_delete.New(s, ps).Handle
	o.handlers.request.deleteAll = requests_delete_all.New(s, ps).Handle
	o.handlers.version.latest = version_latest.New(latestVersionGetter).Handle

	return &o
}

// ApiSessionCreate handles POST /api/session.
func (o *OpenAPI) ApiSessionCreate(w http.ResponseWriter, r *http.Request) {
	payload, err := j.Decode[openapi.CreateSessionRequest](r.Body)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	if err = payload.Validate(); err != nil {
		o.handleError(w, r, err)

		return
	}

	resp, err := o.handlers.session.create(r.Context(), *payload)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusCreated, resp)
}

// ApiSessionCheckExists handles POST /api/session/check/exists.
func (o *OpenAPI) ApiSessionCheckExists(w http.ResponseWriter, r *http.Request) {
	ids, err := j.Decode[openapi.CheckSessionExistsRequest](r.Body)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	const minIDsLen, maxIDsLen = 1, 100

	if err = openapi.ValidateUUIDs(*ids, minIDsLen, maxIDsLen); err != nil {
		o.handleError(w, r, err)

		return
	}

	resp, err := o.handlers.session.exists(r.Context(), *ids)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiSessionDelete handles DELETE /api/session/{sID}.
func (o *OpenAPI) ApiSessionDelete(w http.ResponseWriter, r *http.Request, sID openapi.SessionUUIDInPath) {
	if !openapi.IsValidUUID(sID) {
		o.handleError(w, r, openapi.NewErrBadRequest("invalid session ID: "+sID))

		return
	}

	resp, err := o.handlers.session.delete(r.Context(), sID)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiSessionGet handles GET /api/session/{sID}.
func (o *OpenAPI) ApiSessionGet(w http.ResponseWriter, r *http.Request, sID openapi.SessionUUIDInPath) {
	if !openapi.IsValidUUID(sID) {
		o.handleError(w, r, openapi.NewErrBadRequest("invalid session ID: "+sID))

		return
	}

	resp, err := o.handlers.session.get(r.Context(), sID)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiSessionDeleteAllRequests handles DELETE /api/session/{sID}/requests.
func (o *OpenAPI) ApiSessionDeleteAllRequests(w http.ResponseWriter, r *http.Request, sID openapi.SessionUUIDInPath) {
	if !openapi.IsValidUUID(sID) {
		o.handleError(w, r, openapi.NewErrBadRequest("invalid session ID: "+sID))

		return
	}

	resp, err := o.handlers.request.deleteAll(r.Context(), sID)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiSessionListRequests handles GET /api/session/{sID}/requests.
func (o *OpenAPI) ApiSessionListRequests(w http.ResponseWriter, r *http.Request, sID openapi.SessionUUIDInPath) {
	if !openapi.IsValidUUID(sID) {
		o.handleError(w, r, openapi.NewErrBadRequest("invalid session ID: "+sID))

		return
	}

	resp, err := o.handlers.request.list(r.Context(), sID)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiSessionRequestsSubscribe handles WebSocket upgrade for /api/session/{sID}/requests/subscribe.
func (o *OpenAPI) ApiSessionRequestsSubscribe(
	w http.ResponseWriter,
	r *http.Request,
	sID openapi.SessionUUIDInPath,
	_ openapi.ApiSessionRequestsSubscribeParams,
) {
	if !openapi.IsValidUUID(sID) {
		o.handleError(w, r, openapi.NewErrBadRequest("invalid session ID: "+sID))

		return
	}

	if err := o.handlers.request.subscribe(r.Context(), w, r, sID); err != nil {
		o.handleError(w, r, err)

		return
	}
}

// ApiSessionDeleteRequest handles DELETE /api/session/{sID}/requests/{rID}.
func (o *OpenAPI) ApiSessionDeleteRequest(
	w http.ResponseWriter,
	r *http.Request,
	sID openapi.SessionUUIDInPath,
	rID openapi.RequestUUIDInPath,
) {
	switch {
	case !openapi.IsValidUUID(sID):
		o.handleError(w, r, openapi.NewErrBadRequest("invalid session ID: "+sID))

		return
	case !openapi.IsValidUUID(rID):
		o.handleError(w, r, openapi.NewErrBadRequest("invalid request ID: "+rID))

		return
	}

	resp, err := o.handlers.request.delete(r.Context(), sID, rID)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiSessionGetRequest handles GET /api/session/{sID}/requests/{rID}.
func (o *OpenAPI) ApiSessionGetRequest(
	w http.ResponseWriter,
	r *http.Request,
	sID openapi.SessionUUIDInPath,
	rID openapi.RequestUUIDInPath,
) {
	switch {
	case !openapi.IsValidUUID(sID):
		o.handleError(w, r, openapi.NewErrBadRequest("invalid session ID: "+sID))

		return
	case !openapi.IsValidUUID(rID):
		o.handleError(w, r, openapi.NewErrBadRequest("invalid request ID: "+rID))

		return
	}

	resp, err := o.handlers.request.get(r.Context(), sID, rID)
	if err != nil {
		o.handleError(w, r, err)

		return
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiSettings handles GET /api/settings.
func (o *OpenAPI) ApiSettings(w http.ResponseWriter, r *http.Request) {
	resp := openapi.SettingsResponse{}

	resp.Limits.MaxRequestBodySize = o.settings.MaxRequestBodySize
	resp.Limits.MaxRequests = o.settings.MaxRequests
	resp.Limits.SessionTtl = uint32(o.sessionTTL.Seconds())

	if o.settings.PublicUrlRoot != "" {
		resp.PublicUrlRoot = &o.settings.PublicUrlRoot
	}

	if o.settings.TunnelUrl != nil {
		if tunUrl := o.settings.TunnelUrl.Load(); tunUrl != nil && *tunUrl != "" {
			resp.Tunnel.Enabled, resp.Tunnel.Url = true, tunUrl
		}
	}

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

// ApiAppVersion handles GET /api/version.
func (o *OpenAPI) ApiAppVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=10")

	o.respondWithJSON(w, r, http.StatusOK, openapi.VersionResponse{
		Version: appmeta.Version(),
	})
}

// ApiAppVersionLatest handles GET /api/version/latest.
func (o *OpenAPI) ApiAppVersionLatest(w http.ResponseWriter, r *http.Request) {
	resp, err := o.handlers.version.latest(r.Context())
	if err != nil {
		o.handleError(w, r, err)
		o.log.Warn("failed to get the latest version", logger.Error(err))

		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")

	o.respondWithJSON(w, r, http.StatusOK, resp)
}

const (
	contentTypeHeader   = "Content-Type"
	textPlain, textJson = "text/plain; charset=utf-8", "application/json; charset=utf-8"
)

// LivenessProbe handles GET /healthz.
func (o *OpenAPI) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeHeader, textPlain)
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("OK")); err != nil {
		o.log.Warn("failed to write liveness probe response", logger.Error(err))
	}
}

// LivenessProbeHead handles HEAD /healthz.
func (o *OpenAPI) LivenessProbeHead(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(contentTypeHeader, textPlain)
	w.WriteHeader(http.StatusOK)
}

// ReadinessProbe handles GET /ready.
func (o *OpenAPI) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeHeader, textPlain)

	status, body := http.StatusOK, []byte("OK")

	if err := o.readyChecker(r.Context()); err != nil {
		status, body = http.StatusServiceUnavailable, []byte("readiness probe failed")

		o.log.Warn("readiness probe failed", logger.Error(err))
	}

	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		o.log.Warn("failed to write readiness probe response", logger.Error(err))
	}
}

// ReadinessProbeHead handles HEAD /ready.
func (o *OpenAPI) ReadinessProbeHead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentTypeHeader, textPlain)

	status := http.StatusOK

	if err := o.readyChecker(r.Context()); err != nil {
		status = http.StatusServiceUnavailable

		o.log.Warn("readiness probe failed", logger.Error(err))
	}

	w.WriteHeader(status)
}

// --------------------------------------------------------------------------------------------------------------------

// HandleInternalError is a default error handler for internal server errors (e.g. query parameters binding
// errors, and so on).
func (o *OpenAPI) HandleInternalError(w http.ResponseWriter, r *http.Request, err error) {
	o.handleError(w, r, err)
}

// HandleNotFoundError is a default error handler for "404: not found" errors (scoped for private API, when it
// possible).
func (o *OpenAPI) HandleNotFoundError(w http.ResponseWriter, r *http.Request) {
	o.respondWithJSON(w, r, http.StatusNotFound, openapi.ErrorResponse{
		Error: "handler not found",
	})
}

// --------------------------------------------------------------------------------------------------------------------

func (o *OpenAPI) handleError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		status int
		resp   = openapi.ErrorResponse{}
	)

	switch {
	case openapi.IsOpenAPIError(err) ||
		errors.Is(err, openapi.ErrValidationFailed) ||
		errors.Is(err, j.ErrInvalidJSON) ||
		errors.Is(err, openapi.ErrBadRequest): // 400 Bad Request
		status, resp.Error = http.StatusBadRequest, err.Error()
	case errors.Is(err, openapi.ErrNotFound): // 404 Not Found
		status, resp.Error = http.StatusNotFound, err.Error()
	case errors.Is(err, openapi.ErrServerError): // 500 Internal Server Error
		status, resp.Error = http.StatusInternalServerError, err.Error()
	default: // any other error is treated as an internal server error
		status = http.StatusInternalServerError
		resp.Error = fmt.Sprintf("unexpected error: %v", err) // %v is used to avoid potential panic when err == nil
	}

	o.respondWithJSON(w, r, status, resp)
}

func (o *OpenAPI) respondWithJSON(w http.ResponseWriter, r *http.Request, statusCode int, payload any) {
	w.Header().Set(contentTypeHeader, textJson)
	w.WriteHeader(statusCode)

	if payload == nil || r.Method == http.MethodHead {
		return
	}

	if err := j.EncodeTo(w, payload); err != nil {
		o.log.Warn("failed to encode error response", logger.Error(err))
	}
}
