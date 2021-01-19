// Package panic contains middleware for panics (inside HTTP handlers) logging using "zap" package.
package panic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type response struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

const statusCode = http.StatusInternalServerError

// New creates mux.MiddlewareFunc for panics (inside HTTP handlers) logging using "zap" package. Also it allows
// to respond with JSON-formatted error string instead empty response.
func New(log *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// convert panic reason into error
					err, ok := rec.(error)
					if !ok {
						err = fmt.Errorf("%v", rec)
					}

					stackBuf := make([]byte, 1024)
					// do NOT use `debug.Stack()` here for skipping one unimportant call trace in stacktrace
					for {
						n := runtime.Stack(stackBuf, false)
						if n < len(stackBuf) {
							stackBuf = stackBuf[:n]

							break
						}

						stackBuf = make([]byte, 2*len(stackBuf)) //nolint:gomnd
					}

					// log error with logger
					log.Error("HTTP handler panic", zap.Error(err), zap.String("stacktrace", string(stackBuf)))

					resp := response{
						Message: fmt.Sprintf("%s: %s", http.StatusText(statusCode), err.Error()),
						Code:    statusCode,
					}

					w.WriteHeader(statusCode)

					// and respond with JSON (not "empty response")
					if e := json.NewEncoder(w).Encode(resp); e != nil {
						panic(e)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
