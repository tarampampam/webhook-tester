package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

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
	appVersion "gh.tarampamp.am/webhook-tester/v2/internal/version"
)

type ( // type aliases for better readability
	sID  = openapi.SessionUUIDInPath
	rID  = openapi.RequestUUIDInPath
	skip = openapi.ApiSessionRequestsSubscribeParams
)

type OpenAPI struct {
	log *zap.Logger

	handlers struct {
		settingsGet        func() openapi.SettingsResponse
		sessionCreate      func(openapi.CreateSessionRequest) (*openapi.SessionOptionsResponse, error)
		sessionGet         func(sID) (*openapi.SessionOptionsResponse, error)
		sessionDelete      func(sID) (*openapi.SuccessfulOperationResponse, error)
		requestsList       func(sID) (*openapi.CapturedRequestsListResponse, error)
		requestsDelete     func(sID) (*openapi.SuccessfulOperationResponse, error)
		requestsSubscribe  func(http.ResponseWriter, *http.Request, sID) error
		requestGet         func(sID, rID) (*openapi.CapturedRequestsResponse, error)
		requestDelete      func(sID, rID) (*openapi.SuccessfulOperationResponse, error)
		appVersion         func() openapi.VersionResponse
		appVersionLatest   func(http.ResponseWriter) (*openapi.VersionResponse, error)
		readinessProbe     func(http.ResponseWriter)
		readinessProbeHead func(http.ResponseWriter)
		livenessProbe      func(http.ResponseWriter)
		livenessProbeHead  func(http.ResponseWriter)
	}
}

var _ openapi.ServerInterface = (*OpenAPI)(nil) // verify interface implementation

func NewOpenAPI(ctx context.Context, log *zap.Logger) *OpenAPI {
	var si = &OpenAPI{log: log}

	si.handlers.settingsGet = settings_get.New().Handle
	si.handlers.sessionCreate = session_create.New().Handle
	si.handlers.sessionGet = session_get.New().Handle
	si.handlers.sessionDelete = session_delete.New().Handle
	si.handlers.requestsList = requests_list.New().Handle
	si.handlers.requestsDelete = requests_delete_all.New().Handle
	si.handlers.requestsSubscribe = requests_subscribe.New().Handle
	si.handlers.requestGet = request_get.New().Handle
	si.handlers.requestDelete = request_delete.New().Handle
	si.handlers.appVersion = version.New(appVersion.Version()).Handle
	si.handlers.appVersionLatest = version_latest.New(func() (string, error) { return appVersion.Latest(ctx) }).Handle
	si.handlers.readinessProbe = ready.New().HandleGet
	si.handlers.readinessProbeHead = ready.New().HandleHead
	si.handlers.livenessProbe = live.New().HandleGet
	si.handlers.livenessProbeHead = live.New().HandleHead

	return si
}

func (o *OpenAPI) ApiSettings(w http.ResponseWriter, _ *http.Request) {
	o.respToJson(w, o.handlers.settingsGet())
}

func (o *OpenAPI) ApiSessionCreate(w http.ResponseWriter, r *http.Request) {
	panic("implement me")
}

func (o *OpenAPI) ApiSessionGet(w http.ResponseWriter, _ *http.Request, sID sID) {
	if resp, err := o.handlers.sessionGet(sID); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionDelete(w http.ResponseWriter, _ *http.Request, sID sID) {
	if resp, err := o.handlers.sessionDelete(sID); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionListRequests(w http.ResponseWriter, _ *http.Request, sID sID) {
	if resp, err := o.handlers.requestsList(sID); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionDeleteAllRequests(w http.ResponseWriter, _ *http.Request, sID sID) {
	if resp, err := o.handlers.requestsDelete(sID); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionRequestsSubscribe(w http.ResponseWriter, r *http.Request, sID sID, _ skip) {
	if err := o.handlers.requestsSubscribe(w, r, sID); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	}
}

func (o *OpenAPI) ApiSessionGetRequest(w http.ResponseWriter, _ *http.Request, sID sID, rID rID) {
	if resp, err := o.handlers.requestGet(sID, rID); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiSessionDeleteRequest(w http.ResponseWriter, _ *http.Request, sID sID, rID rID) {
	if resp, err := o.handlers.requestDelete(sID, rID); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ApiAppVersion(w http.ResponseWriter, _ *http.Request) {
	o.respToJson(w, o.handlers.appVersion())
}

func (o *OpenAPI) ApiAppVersionLatest(w http.ResponseWriter, _ *http.Request) {
	if resp, err := o.handlers.appVersionLatest(w); err != nil {
		o.errorToJson(w, err, http.StatusInternalServerError)
	} else {
		o.respToJson(w, resp)
	}
}

func (o *OpenAPI) ReadinessProbe(w http.ResponseWriter, _ *http.Request) {
	o.handlers.readinessProbe(w)
}

func (o *OpenAPI) ReadinessProbeHead(w http.ResponseWriter, _ *http.Request) {
	o.handlers.readinessProbeHead(w)
}

func (o *OpenAPI) LivenessProbe(w http.ResponseWriter, _ *http.Request) {
	o.handlers.livenessProbe(w)
}

func (o *OpenAPI) LivenessProbeHead(w http.ResponseWriter, _ *http.Request) {
	o.handlers.livenessProbeHead(w)
}

// -------------------------------------------------- Error handlers --------------------------------------------------

// HandleInternalError is a default error handler for internal server errors (e.g. query parameters binding
// errors, and so on).
func (o *OpenAPI) HandleInternalError(w http.ResponseWriter, _ *http.Request, err error) {
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
