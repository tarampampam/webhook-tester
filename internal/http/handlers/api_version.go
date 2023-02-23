package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"gh.tarampamp.am/webhook-tester/internal/api"
)

type apiVersion struct {
	version string
}

// ApiAppVersion returns application version.
func (s *apiVersion) ApiAppVersion(c echo.Context) error {
	return c.JSON(http.StatusOK, api.AppVersion{Version: s.version})
}
