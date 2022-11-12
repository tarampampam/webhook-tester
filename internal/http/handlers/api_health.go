package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type checker interface {
	Check() error
}

type apiHealth struct {
	liveChecker, readyChecker checker
}

func (s *apiHealth) makeCheck(c echo.Context, chk checker) error {
	if err := chk.Check(); err != nil {
		return c.String(http.StatusServiceUnavailable, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

// LivenessProbe returns code 200 if the application is alive.
func (s *apiHealth) LivenessProbe(c echo.Context) error {
	return s.makeCheck(c, s.liveChecker)
}

// ReadinessProbe returns code 200 if the application is ready to serve traffic.
func (s *apiHealth) ReadinessProbe(c echo.Context) error {
	return s.makeCheck(c, s.readyChecker)
}
