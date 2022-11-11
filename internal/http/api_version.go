package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/version"
)

type apiVersion struct{}

func (a *apiVersion) ApiAppVersion(c echo.Context) error {
	return c.JSON(http.StatusOK, api.AppVersion{Version: version.Version()})
}
