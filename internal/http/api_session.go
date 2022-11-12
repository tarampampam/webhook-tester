package http

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type apiSession struct {
	storage storage.Storage
}

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
