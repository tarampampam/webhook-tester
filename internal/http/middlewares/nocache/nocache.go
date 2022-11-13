// Package nocache contains middleware for HTTP response caching disabling.
package nocache

import "github.com/labstack/echo/v4"

// New creates echo.MiddlewareFunc for HTTP response caching disabling.
func New() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Response().Header().Set("Pragma", "no-cache")
			c.Response().Header().Set("Expires", "0")

			return next(c)
		}
	}
}
