// Package panic contains middleware for panics (inside HTTP handlers) logging using "zap" package.
package panic

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type response struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

const statusCode = http.StatusInternalServerError

// New creates mux.MiddlewareFunc for panics (inside HTTP handlers) logging using "zap" package. Also it allows
// to respond with JSON-formatted error string instead empty response.
func New(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if rec := recover(); rec != nil {
					// convert panic reason into error
					err, ok := rec.(error)
					if !ok {
						err = fmt.Errorf("%v", rec)
					}

					stackBuf := make([]byte, 1024) //nolint:mnd

					// do NOT use `debug.Stack()` here for skipping one unimportant call trace in stacktrace
					for {
						n := runtime.Stack(stackBuf, false)
						if n < len(stackBuf) {
							stackBuf = stackBuf[:n]

							break
						}

						stackBuf = make([]byte, 2*len(stackBuf)) //nolint:mnd
					}

					// log error with logger
					log.Error("HTTP handler panic", zap.Error(err), zap.String("stacktrace", string(stackBuf)))

					if respErr := c.JSON(statusCode, response{ // and respond with JSON (not "empty response")
						Message: fmt.Sprintf("%s: %s", http.StatusText(statusCode), err.Error()),
						Code:    statusCode,
					}); respErr != nil {
						panic(respErr)
					}
				}
			}()

			return next(c)
		}
	}
}
