package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"gh.tarampamp.am/webhook-tester/v2/internal/config"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/live"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/ready"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/request_delete"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/request_get"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/requests_delete_all"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/requests_list"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/requests_subscribe"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/session_create"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/session_delete"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/session_get"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/settings_get"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/version"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/version_latest"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
	appVersion "gh.tarampamp.am/webhook-tester/v2/internal/version"
)

type ( // type aliases for better readability
	sID  = openapi.SessionUUIDInPath
	rID  = openapi.RequestUUIDInPath
	skip = openapi.ApiSessionRequestsSubscribeParams // it doesn't matter
)

type OpenAPI struct {
	log *zap.Logger

	handlers struct {
		settingsGet       func() openapi.SettingsResponse
		sessionCreate     func(context.Context, openapi.CreateSessionRequest) (*openapi.SessionOptionsResponse, error)
		sessionGet        func(context.Context, sID) (*openapi.SessionOptionsResponse, error)
		sessionDelete     func(context.Context, sID) (*openapi.SuccessfulOperationResponse, error)
		requestsList      func(context.Context, sID) (*openapi.CapturedRequestsListResponse, error)
		requestsDelete    func(context.Context, sID) (*openapi.SuccessfulOperationResponse, error)
		requestsSubscribe func(context.Context, http.ResponseWriter, *http.Request, sID) error
		requestGet        func(context.Context, sID, rID) (*openapi.CapturedRequestsResponse, error)
		requestDelete     func(context.Context, sID, rID) (*openapi.SuccessfulOperationResponse, error)
		appVersion        func() openapi.VersionResponse
		appVersionLatest  func(context.Context, http.ResponseWriter) (*openapi.VersionResponse, error)
		readinessProbe    func(context.Context, http.ResponseWriter, string)
		livenessProbe     func(http.ResponseWriter, string)
	}
}

var _ openapi.ServerInterface = (*OpenAPI)(nil) // verify interface implementation

func NewOpenAPI(
	log *zap.Logger,
	rdyChecker func(context.Context) error,
	lastAppVer func(context.Context) (string, error),
	cfg config.AppSettings,
	db storage.Storage,
	pubSub pubsub.PubSub[pubsub.CapturedRequest],
) *OpenAPI {
	var si = &OpenAPI{log: log}

	si.handlers.settingsGet = settings_get.New(cfg).Handle
	si.handlers.sessionCreate = session_create.New(db).Handle
	si.handlers.sessionGet = session_get.New(db).Handle
	si.handlers.sessionDelete = session_delete.New(db).Handle
	si.handlers.requestsList = requests_list.New(db).Handle
	si.handlers.requestsDelete = requests_delete_all.New(db).Handle
	si.handlers.requestsSubscribe = requests_subscribe.New(db, pubSub).Handle
	si.handlers.requestGet = request_get.New(db).Handle
	si.handlers.requestDelete = request_delete.New(db).Handle
	si.handlers.appVersion = version.New(appVersion.Version()).Handle
	si.handlers.appVersionLatest = version_latest.New(lastAppVer).Handle
	si.handlers.readinessProbe = ready.New(rdyChecker).Handle
	si.handlers.livenessProbe = live.New().Handle

	return si
}

func (o *OpenAPI) ApiSettings(w http.ResponseWriter, _ *http.Request) {
	o.respToJson(w, o.handlers.settingsGet())
}

func (o *OpenAPI) ApiSessionCreate(w http.ResponseWriter, r *http.Request) {
	var payload openapi.CreateSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		o.errorToJson(w, err, http.StatusBadRequest)

		return
	}

	if err := payload.Validate(); err != nil {
		o.errorToJson(w, err, http.StatusBadRequest)

		return
	}

	if resp, err := o.handlers.sessionCreate(r.Context(), payload); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionGet(w http.ResponseWriter, r *http.Request, sID sID) {
	if resp, err := o.handlers.sessionGet(r.Context(), sID); err != nil {
		var statusCode = http.StatusInternalServerError

		if errors.Is(err, storage.ErrSessionNotFound) {
			statusCode = http.StatusNotFound
		}

		o.errorToJson(w, err, statusCode)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionDelete(w http.ResponseWriter, r *http.Request, sID sID) {
	if resp, err := o.handlers.sessionDelete(r.Context(), sID); err != nil {
		var statusCode = http.StatusInternalServerError

		if errors.Is(err, storage.ErrSessionNotFound) {
			statusCode = http.StatusNotFound
		}

		o.errorToJson(w, err, statusCode)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionListRequests(w http.ResponseWriter, r *http.Request, sID sID) {
	if resp, err := o.handlers.requestsList(r.Context(), sID); err != nil {
		var statusCode = http.StatusInternalServerError

		if errors.Is(err, storage.ErrSessionNotFound) {
			statusCode = http.StatusNotFound
		}

		o.errorToJson(w, err, statusCode)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionDeleteAllRequests(w http.ResponseWriter, r *http.Request, sID sID) {
	if resp, err := o.handlers.requestsDelete(r.Context(), sID); err != nil {
		var statusCode = http.StatusInternalServerError

		if errors.Is(err, storage.ErrSessionNotFound) {
			statusCode = http.StatusNotFound
		}

		o.errorToJson(w, err, statusCode)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionRequestsSubscribe(w http.ResponseWriter, r *http.Request, sID sID, _ skip) {
	if err := o.handlers.requestsSubscribe(r.Context(), w, r, sID); err != nil {
		var statusCode = http.StatusInternalServerError

		if errors.Is(err, storage.ErrSessionNotFound) {
			statusCode = http.StatusNotFound
		}

		o.errorToJson(w, err, statusCode)
	}
}

func (o *OpenAPI) ApiSessionGetRequest(w http.ResponseWriter, r *http.Request, sID sID, rID rID) {
	if resp, err := o.handlers.requestGet(r.Context(), sID, rID); err != nil {
		var statusCode = http.StatusInternalServerError

		if errors.Is(err, storage.ErrNotFound) {
			statusCode = http.StatusNotFound
		}

		o.errorToJson(w, err, statusCode)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionDeleteRequest(w http.ResponseWriter, r *http.Request, sID sID, rID rID) {
	if resp, err := o.handlers.requestDelete(r.Context(), sID, rID); err != nil {
		var statusCode = http.StatusInternalServerError

		if errors.Is(err, storage.ErrNotFound) {
			statusCode = http.StatusNotFound
		}

		o.errorToJson(w, err, statusCode)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiAppVersion(w http.ResponseWriter, _ *http.Request) {
	o.respToJson(w, o.handlers.appVersion())
}

func (o *OpenAPI) ApiAppVersionLatest(w http.ResponseWriter, r *http.Request) {
	if resp, err := o.handlers.appVersionLatest(r.Context(), w); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	o.handlers.readinessProbe(r.Context(), w, r.Method)
}

func (o *OpenAPI) ReadinessProbeHead(w http.ResponseWriter, r *http.Request) {
	o.handlers.readinessProbe(r.Context(), w, r.Method)
}

func (o *OpenAPI) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	o.handlers.livenessProbe(w, r.Method)
}

func (o *OpenAPI) LivenessProbeHead(w http.ResponseWriter, r *http.Request) {
	o.handlers.livenessProbe(w, r.Method)
}

// -------------------------------------------------- Error handlers --------------------------------------------------

// HandleInternalError is a default error handler for internal server errors (e.g. query parameters binding
// errors, and so on).
func (o *OpenAPI) HandleInternalError(w http.ResponseWriter, _ *http.Request, err error) {
	//	Invalid format for parameter session_uuid: error unmarshaling 'xxxxxx' text as *uuid.UUID: invalid UUID format
	// to
	//	invalid UUID format
	if err != nil && strings.Contains(err.Error(), "invalid UUID") {
		err = errors.New("invalid UUID format")
	}

	o.errorToJson(w, err, http.StatusBadRequest)
}

// HandleNotFoundError is a default error handler for "404: not found" errors.
func (o *OpenAPI) HandleNotFoundError(w http.ResponseWriter, _ *http.Request) {
	o.errorToJson(w, errors.New("not found"), http.StatusNotFound)
}

// ------------------------------------------------- Internal helpers -------------------------------------------------

const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json; charset=utf-8"
)

func (o *OpenAPI) respToJson(w http.ResponseWriter, resp any) {
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(http.StatusOK)

	if resp == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		o.log.Error("failed to encode/write response", zap.Error(err))
	}
}

func (o *OpenAPI) errorToJson(w http.ResponseWriter, err error, status int) {
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(status)

	if err == nil {
		return
	}

	if encErr := json.NewEncoder(w).Encode(openapi.ErrorResponse{Error: err.Error()}); encErr != nil {
		o.log.Error("failed to encode/write error response", zap.Error(encErr))
	}
}
