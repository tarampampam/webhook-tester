package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type apiSession struct {
	storage storage.Storage
	pub     pubsub.Publisher
}

// ApiSessionCreate creates a new session with passed parameters.
func (s *apiSession) ApiSessionCreate(c echo.Context) error {
	var payload api.NewSession

	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, api.BadRequest{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
	}

	if err := payload.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, api.BadRequest{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
	}

	var (
		content     = payload.ResponseContent()
		status      = payload.GetStatusCode()
		contentType = payload.GetContentType()
		delay       = payload.GetResponseDelay()
	)

	sessionUuid, err := s.storage.CreateSession(content, status, contentType, time.Second*time.Duration(delay))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	}

	u, _ := uuid.Parse(sessionUuid)

	return c.JSON(http.StatusOK, api.SessionOptions{
		CreatedAtUnix: int(time.Now().Unix()),
		Response: api.SessionResponseOptions{
			Code:          api.StatusCode(status),
			ContentBase64: base64.StdEncoding.EncodeToString(content),
			ContentType:   contentType,
			DelaySec:      delay,
		},
		Uuid: u,
	})
}

// ApiSessionDelete deletes the session with the passed UUID (and all associated requests).
func (s *apiSession) ApiSessionDelete(c echo.Context, session api.SessionUUID) error {
	// delete session
	if result, err := s.storage.DeleteSession(session.String()); err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	} else if !result {
		return c.JSON(http.StatusNotFound, api.NotFound{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("the session with UUID %s was not found", session),
		})
	}

	// and recorded session requests
	if _, err := s.storage.DeleteRequests(session.String()); err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, api.SuccessfulOperation{
		Success: true,
	})
}

// ApiSessionDeleteAllRequests deletes all recorded session requests.
func (s *apiSession) ApiSessionDeleteAllRequests(c echo.Context, session api.SessionUUID) error {
	if deleted, err := s.storage.DeleteRequests(session.String()); err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	} else if !deleted {
		return c.JSON(http.StatusNotFound, api.ServerError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("requests for the session with UUID %s was not found", session.String()),
		})
	}

	go func() { _ = s.pub.Publish(session.String(), pubsub.NewAllRequestsDeletedEvent()) }()

	return c.JSON(http.StatusOK, api.SuccessfulOperation{
		Success: true,
	})
}

func (s *apiSession) convertStoredRequestToApiStruct(in storage.Request) api.SessionRequest {
	var (
		headersMap = in.Headers()
		headers    = make([]api.HttpHeader, 0, len(headersMap))
	)

	for name, value := range headersMap {
		headers = append(headers, api.HttpHeader{Name: name, Value: value})
	}

	sort.SliceStable(headers, func(j, k int) bool { return headers[j].Name < headers[k].Name })

	u, _ := uuid.Parse(in.UUID())

	return api.SessionRequest{
		Uuid:          u,
		ClientAddress: in.ClientAddr(),
		Method:        api.HttpMethod(in.Method()),
		ContentBase64: base64.StdEncoding.EncodeToString(in.Content()),
		Headers:       headers,
		Url:           in.URI(),
		CreatedAtUnix: api.UnixTime(in.CreatedAt().Unix()),
	}
}

// ApiSessionGetAllRequests returns all session recorded requests.
func (s *apiSession) ApiSessionGetAllRequests(c echo.Context, sessionUuid api.SessionUUID) error {
	if session, err := s.storage.GetSession(sessionUuid.String()); session == nil {
		if err != nil {
			return c.JSON(http.StatusInternalServerError, api.ServerError{
				Code:    http.StatusInternalServerError,
				Message: errors.Wrap(err, "cannot read session data").Error(),
			})
		}

		return c.JSON(http.StatusNotFound, api.ServerError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("the session with UUID %s was not found", sessionUuid.String()),
		})
	}

	requests, err := s.storage.GetAllRequests(sessionUuid.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: errors.Wrap(err, "cannot get requests data").Error(),
		})
	}

	var result = make([]api.SessionRequest, 0, len(requests))

	for i := 0; i < len(requests); i++ {
		result = append(result, s.convertStoredRequestToApiStruct(requests[i]))
	}

	// sort requests from newest to oldest
	sort.SliceStable(result, func(j, k int) bool { return result[j].CreatedAtUnix > result[k].CreatedAtUnix })

	return c.JSON(http.StatusOK, result)
}

// ApiSessionDeleteRequest deletes the request with passed UUID.
func (s *apiSession) ApiSessionDeleteRequest(c echo.Context, session api.SessionUUID, request api.RequestUUID) error {
	if deleted, err := s.storage.DeleteRequest(session.String(), request.String()); err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: errors.Wrap(err, "cannot delete the request").Error(),
		})
	} else if !deleted {
		return c.JSON(http.StatusNotFound, api.ServerError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("request with UUID %s was not found", request.String()),
		})
	}

	go func() { _ = s.pub.Publish(session.String(), pubsub.NewRequestDeletedEvent(request.String())) }()

	return c.JSON(http.StatusOK, api.SuccessfulOperation{
		Success: true,
	})
}

func (s *apiSession) ApiSessionGetRequest(c echo.Context, session api.SessionUUID, requestUuid api.RequestUUID) error {
	request, err := s.storage.GetRequest(session.String(), requestUuid.String())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: errors.Wrap(err, "cannot read request data").Error(),
		})
	} else if request == nil {
		return c.JSON(http.StatusNotFound, api.ServerError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("request with UUID %s was not found", requestUuid.String()),
		})
	}

	return c.JSON(http.StatusOK, s.convertStoredRequestToApiStruct(request))
}
