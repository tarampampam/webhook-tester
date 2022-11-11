package http

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

func (s *apiHealth) LivenessProbe(c echo.Context) error {
	return s.makeCheck(c, s.liveChecker)
}

func (s *apiHealth) ReadinessProbe(c echo.Context) error {
	return s.makeCheck(c, s.readyChecker)
}
