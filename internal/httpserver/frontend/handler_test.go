package frontend_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/frontend"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	root := fs.FS(fstest.MapFS{
		"index.html": {
			Data: []byte("<html><body>index</body></html>"),
		},
		"index.html.gz": {
			Data: []byte("GZIP_INDEX"),
		},
		"robots.txt": {
			Data: []byte("User-agent: *\nDisallow: /"),
		},
		"robots.txt.gz": {
			Data: []byte("GZIP_ROBOTS"),
		},
		"dir/file.txt": {
			Data: []byte("file content"),
		},
		"dir/file.txt.gz": {
			Data: []byte("GZIP_FILE"),
		},
		"styles.css": { // no .gz counterpart — used to test gzip-missing behavior
			Data: []byte("body{color:red}"),
		},
	})

	for name, tt := range map[string]struct {
		giveRoot              fs.FS
		giveURL               string
		giveMethod            string
		giveHeaders           map[string]string
		wantCode              int
		wantInBody            string
		wantEmptyResponseBody bool
		wantHeaders           map[string]string
	}{
		// basic behavior
		"root": {
			giveRoot:   root,
			giveURL:    "/",
			giveMethod: http.MethodGet,
			wantCode:   http.StatusOK,
			wantInBody: "<body>index</body>",
			wantHeaders: map[string]string{
				"Content-Type":   "text/html; charset=utf-8",
				"Content-Length": "31",
				"ETag":           `"f995004c079c07e3"`,
			},
		},
		"root (head)": {
			giveRoot:              root,
			giveURL:               "/",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
			wantHeaders: map[string]string{
				"Content-Type":   "text/html; charset=utf-8",
				"Content-Length": "31",
				"ETag":           `"f995004c079c07e3"`,
			},
		},
		"index redirect": {
			giveRoot:              root,
			giveURL:               "/index.html",
			giveMethod:            http.MethodGet,
			wantCode:              http.StatusMovedPermanently,
			wantEmptyResponseBody: true,
			wantHeaders:           map[string]string{"Location": "/"},
		},
		"index redirect (head)": {
			giveRoot:              root,
			giveURL:               "/index.html",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusMovedPermanently,
			wantEmptyResponseBody: true,
			wantHeaders:           map[string]string{"Location": "/"},
		},
		"not found -> SPA fallback (index.html)": {
			giveRoot:   root,
			giveURL:    "/foo",
			giveMethod: http.MethodGet,
			wantCode:   http.StatusOK,
			wantInBody: "<html><body>index</body></html>",
			wantHeaders: map[string]string{
				"Content-Type":   "text/html; charset=utf-8",
				"Content-Length": "31",
				"ETag":           `"f995004c079c07e3"`,
			},
		},
		"not found (head)": {
			giveRoot:              root,
			giveURL:               "/foo",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK, // <-- IMPORTANT
			wantEmptyResponseBody: true,
			wantHeaders: map[string]string{
				"Content-Type":   "text/html; charset=utf-8",
				"Content-Length": "31",
				"ETag":           `"f995004c079c07e3"`,
			},
		},
		"existing file (head)": {
			giveRoot:              root,
			giveURL:               "/robots.txt",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusOK, // <-- IMPORTANT
			wantEmptyResponseBody: true,          // <-- IMPORTANT
			wantHeaders: map[string]string{
				"Content-Type":   "text/plain; charset=utf-8",
				"Content-Length": "25",
			},
		},
		"fallback404": {
			giveRoot:    fs.FS(fstest.MapFS{}),
			giveURL:     "/",
			giveMethod:  http.MethodGet,
			wantCode:    http.StatusNotFound, // <-- IMPORTANT
			wantInBody:  "<title>Not found</title>",
			wantHeaders: map[string]string{"Content-Type": "text/html; charset=utf-8"},
		},
		"fallback404 (head)": {
			giveRoot:              fs.FS(fstest.MapFS{}),
			giveURL:               "/",
			giveMethod:            http.MethodHead,
			wantCode:              http.StatusNotFound, // <-- IMPORTANT
			wantEmptyResponseBody: true,                // <-- IMPORTANT
		},
		"fallback404 gzip not served": {
			giveRoot:    fs.FS(fstest.MapFS{}),
			giveURL:     "/",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "gzip"},
			wantCode:    http.StatusNotFound, // <-- IMPORTANT
			wantInBody:  "<title>Not found</title>",
			wantHeaders: map[string]string{"Content-Encoding": ""}, // <-- IMPORTANT
		},
		"path traversal neutralized": {
			// /../etc/passwd cleans to /etc/passwd -> not in FS -> SPA fallback
			giveRoot:   root,
			giveURL:    "/../etc/passwd",
			giveMethod: http.MethodGet,
			wantCode:   http.StatusOK,
			wantInBody: "<html><body>index</body></html>",
		},
		"POST request served same as GET": {
			giveRoot:   root,
			giveURL:    "/robots.txt",
			giveMethod: http.MethodPost,
			wantCode:   http.StatusOK,
			wantInBody: "User-agent",
		},
		"content-type sniffed from content when no extension": {
			// no file extension -> mime.TypeByExtension("") == "" -> handler reads first 512 bytes
			// via http.DetectContentType; `<?xml` prefix is detected as text/xml
			giveRoot: fs.FS(fstest.MapFS{
				"icon": {
					Data: []byte(
						`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="10" ` +
							`height="10"><circle cx="5" cy="5" r="4"/></svg>`,
					),
				},
			}),
			giveURL:     "/icon",
			giveMethod:  http.MethodGet,
			wantCode:    http.StatusOK,
			wantInBody:  "<svg",
			wantHeaders: map[string]string{"Content-Type": "text/xml; charset=utf-8"},
		},

		// gzip behavior
		"gzip index served": {
			giveRoot:    root,
			giveURL:     "/",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "gzip"},
			wantCode:    http.StatusOK,
			wantInBody:  "GZIP_INDEX",
			wantHeaders: map[string]string{
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Content-Type":     "text/html; charset=utf-8",
				"Content-Length":   "10",
				"ETag":             `"b1f7f97bb0b53962"`,
			},
		},
		"gzip robots served": {
			giveRoot:    root,
			giveURL:     "/robots.txt",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "gzip"},
			wantCode:    http.StatusOK,
			wantInBody:  "GZIP_ROBOTS",
			wantHeaders: map[string]string{
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Content-Type":     "text/plain; charset=utf-8",
				"Content-Length":   "11",
				"ETag":             `"8392b4ebca2933d3"`,
			},
		},
		"gzip HEAD": {
			giveRoot:              root,
			giveURL:               "/robots.txt",
			giveMethod:            http.MethodHead,
			giveHeaders:           map[string]string{"Accept-Encoding": "gzip"},
			wantCode:              http.StatusOK,
			wantEmptyResponseBody: true,
			wantHeaders: map[string]string{
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Content-Type":     "text/plain; charset=utf-8",
				"Content-Length":   "11",
			},
		},
		"gzip disabled via q=0": {
			giveRoot:    root,
			giveURL:     "/robots.txt",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "gzip;q=0"},
			wantCode:    http.StatusOK,
			wantInBody:  "User-agent",
			wantHeaders: map[string]string{
				"Content-Encoding": "",
				"Content-Type":     "text/plain; charset=utf-8",
				"Content-Length":   "25",
			},
		},
		"gzip fallback when encoding unsupported": {
			giveRoot:    root,
			giveURL:     "/robots.txt",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "br"},
			wantCode:    http.StatusOK,
			wantInBody:  "User-agent",
			wantHeaders: map[string]string{
				"Content-Encoding": "",
				"Content-Type":     "text/plain; charset=utf-8",
				"Content-Length":   "25",
			},
		},
		"gzip nested file": {
			giveRoot:    root,
			giveURL:     "/dir/file.txt",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "gzip"},
			wantCode:    http.StatusOK,
			wantInBody:  "GZIP_FILE",
			wantHeaders: map[string]string{
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Content-Type":     "text/plain; charset=utf-8",
				"Content-Length":   "9",
				"ETag":             `"26af49fa26295478"`,
			},
		},
		"gzip not served when .gz missing": {
			giveRoot:    root,
			giveURL:     "/styles.css",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "gzip"},
			wantCode:    http.StatusOK,
			wantInBody:  "body{color:red}",
			wantHeaders: map[string]string{
				"Content-Encoding": "", // no .gz on disk -> plain content served
				"Content-Type":     "text/css; charset=utf-8",
				"Content-Length":   "15",
			},
		},
		"gzip multiple accept-encoding values": {
			giveRoot:    root,
			giveURL:     "/robots.txt",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "deflate, gzip"},
			wantCode:    http.StatusOK,
			wantInBody:  "GZIP_ROBOTS",
			wantHeaders: map[string]string{
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Content-Type":     "text/plain; charset=utf-8",
				"Content-Length":   "11",
				"ETag":             `"8392b4ebca2933d3"`,
			},
		},
		"SPA fallback gzip not served": {
			// gzip is tied to the requested path - /nonexistent.gz does not exist,
			// so the plain index.html is served even though index.html.gz exists
			giveRoot:    root,
			giveURL:     "/nonexistent",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"Accept-Encoding": "gzip"},
			wantCode:    http.StatusOK,
			wantInBody:  "<html><body>index</body></html>",
			wantHeaders: map[string]string{
				"Content-Encoding": "",
				"Content-Type":     "text/html; charset=utf-8",
				"Content-Length":   "31",
				"ETag":             `"f995004c079c07e3"`,
			},
		},

		// conditional requests (If-None-Match)
		"304 on matching ETag": {
			giveRoot:              root,
			giveURL:               "/robots.txt",
			giveMethod:            http.MethodGet,
			giveHeaders:           map[string]string{"If-None-Match": `"31c44afa7afc1c78"`},
			wantCode:              http.StatusNotModified,
			wantEmptyResponseBody: true,
		},
		"304 on matching ETag (HEAD)": {
			giveRoot:              root,
			giveURL:               "/robots.txt",
			giveMethod:            http.MethodHead,
			giveHeaders:           map[string]string{"If-None-Match": `"31c44afa7afc1c78"`},
			wantCode:              http.StatusNotModified,
			wantEmptyResponseBody: true,
		},
		"200 on mismatched ETag": {
			giveRoot:    root,
			giveURL:     "/robots.txt",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"If-None-Match": `"deadbeef"`},
			wantCode:    http.StatusOK,
			wantInBody:  "User-agent",
		},
		"no 304 for fallback404 page": {
			// fallback-404 has an ETag too, but we must not return 304 for it
			giveRoot:    fs.FS(fstest.MapFS{}),
			giveURL:     "/",
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{"If-None-Match": `"any-etag"`},
			wantCode:    http.StatusNotFound,
			wantInBody:  "<title>Not found</title>",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.giveMethod, tt.giveURL, http.NoBody)
			for k, v := range tt.giveHeaders {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()

			frontend.New(tt.giveRoot).ServeHTTP(rec, req)
			assert.Equal(t, tt.wantCode, rec.Code)

			if tt.wantEmptyResponseBody {
				assert.Empty(t, rec.Body.String())
			} else if tt.wantInBody != "" {
				assert.Contains(t, rec.Body.String(), tt.wantInBody)
			}

			for k, v := range tt.wantHeaders {
				assert.Equal(t, v, rec.Header().Get(k))
			}
		})
	}
}
