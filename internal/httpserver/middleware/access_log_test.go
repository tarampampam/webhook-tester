package middleware_test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// TestNewAccessLog is the primary table-driven test for the NewAccessLog middleware.
// Each case provides a pre-built *http.Request, an inner handler, an optional skipper,
// and a checkLogEntry function that receives the captured log records for assertions.
func TestNewAccessLog(t *testing.T) {
	t.Parallel()

	// injectLogger wraps next so that log is always present in the request context, mimicking what NewInjectLog
	// middleware would do in production.
	injectLogger := func(log *logger.Logger, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(logger.With(r.Context(), log)))
		})
	}

	t.Run("common cases", func(t *testing.T) {
		t.Parallel()

		for name, tc := range map[string]struct {
			giveRequest        *http.Request
			giveDefaultLevel   logger.Level
			giveSkipper        func(*http.Request) bool
			giveHandler        http.HandlerFunc
			giveResponseWriter func() http.ResponseWriter // nil → httptest.NewRecorder()
			checkLogEntry      func(t *testing.T, rec *logger.Recorder)
		}{
			"2xx: logged at configured level": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/api/resource?foo=bar", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
					e := rec.Records()[0]
					assert.Equal(t, logger.InfoLevel, e.Level)
					assert.Contains(t, e.Message, "HTTP request processed")
					assert.Equal(t, int64(http.StatusOK), e.Attrs["status_code"].Int64())
					assert.Equal(t, "/api/resource?foo=bar", e.Attrs["uri"].String())
					_, hasDuration := e.Attrs["duration"]
					assert.True(t, hasDuration)
				},
			},
			"3xx: logged at configured level": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/old", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusMovedPermanently) },
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
					e := rec.Records()[0]
					assert.Equal(t, logger.InfoLevel, e.Level)
					assert.Contains(t, e.Message, "HTTP request processed")
					assert.Equal(t, int64(http.StatusMovedPermanently), e.Attrs["status_code"].Int64())
				},
			},
			"4xx: logged at configured level": {
				giveRequest:      httptest.NewRequest(http.MethodPost, "/protected", nil),
				giveDefaultLevel: logger.DebugLevel,
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) },
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
					e := rec.Records()[0]
					assert.Equal(t, logger.DebugLevel, e.Level)
					assert.Contains(t, e.Message, "HTTP request processed")
					assert.Equal(t, int64(http.StatusUnauthorized), e.Attrs["status_code"].Int64())
				},
			},
			"5xx: logged at configured level": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/crash", nil),
				giveDefaultLevel: logger.DebugLevel,
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
					e := rec.Records()[0]
					assert.Equal(t, logger.DebugLevel, e.Level)
					assert.Contains(t, e.Message, "HTTP request processed")
					assert.Equal(t, int64(http.StatusInternalServerError), e.Attrs["status_code"].Int64())
				},
			},
			"implicit 200 when body written without WriteHeader": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/implicit", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveHandler: func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("hello"))
				},
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
					e := rec.Records()[0]
					assert.Equal(t, int64(http.StatusOK), e.Attrs["status_code"].Int64())
					assert.Equal(t, int64(5), e.Attrs["response_size"].Int64())
				},
			},
			"response_size accumulates across multiple Write calls": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/chunked", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveHandler: func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("foo"))
					_, _ = w.Write([]byte("bar"))
					_, _ = w.Write([]byte("baz"))
				},
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
					assert.Equal(t, int64(9), rec.Records()[0].Attrs["response_size"].Int64())
				},
			},
			"all standard request fields are present in log entry": {
				giveRequest: func() *http.Request {
					req := httptest.NewRequest(http.MethodPost, "/api/v1/items", nil)
					req.Header.Set("User-Agent", "test-agent/2.0")
					req.Host = "api.example.com"

					return req
				}(),
				giveDefaultLevel: logger.InfoLevel,
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) },
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
					e := rec.Records()[0]
					assert.Equal(t, int64(http.StatusCreated), e.Attrs["status_code"].Int64())
					assert.Equal(t, "POST", e.Attrs["method"].String())
					assert.Equal(t, "test-agent/2.0", e.Attrs["useragent"].String())
					assert.Equal(t, "api.example.com", e.Attrs["host"].String())
					assert.True(t, e.Attrs["remote_addr"].String() != "")
					_, hasDuration := e.Attrs["duration"]
					assert.True(t, hasDuration)

					_, hasURI := e.Attrs["uri"]
					assert.True(t, hasURI)
					assert.Equal(t, int64(0), e.Attrs["response_size"].Int64())
				},
			},
			"skipper returns true: handler is called but nothing is logged": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/healthz", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveSkipper:      func(_ *http.Request) bool { return true },
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 0, rec.Len())
				},
			},
			"skipper returns false: request is logged normally": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/api", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveSkipper:      func(_ *http.Request) bool { return false },
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					assert.Equal(t, 1, rec.Len())
				},
			},
			"no logger in context: no panic, handler still called": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/no-logger", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveHandler:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				checkLogEntry:    nil,
			},
			"hijack is delegated to underlying ResponseWriter": {
				giveRequest:      httptest.NewRequest(http.MethodGet, "/ws", nil),
				giveDefaultLevel: logger.InfoLevel,
				giveResponseWriter: func() http.ResponseWriter {
					return &hijackableResponseWriter{ResponseRecorder: httptest.NewRecorder()}
				},
				giveHandler: func(w http.ResponseWriter, r *http.Request) {
					hj, ok := w.(http.Hijacker)
					assert.True(t, ok)

					conn, _, err := hj.Hijack()
					assert.NoError(t, err)

					_ = conn.Close()
				},
				checkLogEntry: func(t *testing.T, rec *logger.Recorder) {
					// after Hijack the status is never written, so the middleware
					// logs with the default implicit 200.
					assert.Equal(t, 1, rec.Len())
				},
			},
		} {
			t.Run(name, func(t *testing.T) {
				log, rec := logger.NewRecorder()

				var handlerCalled bool

				inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handlerCalled = true

					if tc.giveHandler != nil {
						tc.giveHandler(w, r)
					}
				})

				mw := middleware.NewAccessLog(tc.giveDefaultLevel, tc.giveSkipper)

				handler := injectLogger(log, mw(inner))
				if name == "no logger in context: no panic, handler still called" {
					handler = mw(inner)
				}

				w := http.ResponseWriter(httptest.NewRecorder())
				if tc.giveResponseWriter != nil {
					w = tc.giveResponseWriter()
				}

				assert.NotPanics(t, func() { handler.ServeHTTP(w, tc.giveRequest) })
				assert.True(t, handlerCalled)

				if tc.checkLogEntry != nil {
					tc.checkLogEntry(t, rec)
				}
			})
		}
	})

	t.Run("doubled invocation guard", func(t *testing.T) {
		t.Parallel()

		log, logRec := logger.NewRecorder()

		mw := middleware.NewAccessLog(logger.InfoLevel, nil)

		// wrap the inner handler twice with the same middleware constructor.
		handler := injectLogger(log, mw(mw(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, 1, logRec.Len())
	})
}

// --------------------------------------------------------------------------------------------------------------------

// hijackableResponseWriter is a test double that adds http.Hijacker on top of httptest.ResponseRecorder so the
// Hijack-delegation path can be exercised.
type hijackableResponseWriter struct {
	*httptest.ResponseRecorder
}

func (h *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c1, c2 := net.Pipe()
	_ = c2.Close()

	return c1, bufio.NewReadWriter(bufio.NewReader(c1), bufio.NewWriter(c1)), nil
}
