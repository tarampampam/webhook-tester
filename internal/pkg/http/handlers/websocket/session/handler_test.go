package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/websocket/session"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type fakeMetrics struct {
	c int
}

func (f *fakeMetrics) IncrementActiveClients() { f.c++ }
func (f *fakeMetrics) DecrementActiveClients() { f.c-- }

func TestHandler_ServeHTTPErrors(t *testing.T) {
	var cases = []struct {
		name           string
		giveReqVars    map[string]string
		wantStatusCode int
		wantContent    string
	}{
		{
			name:           "without registered session UUID",
			giveReqVars:    nil,
			wantStatusCode: http.StatusInternalServerError,
			wantContent:    "cannot extract session UUID",
		},
		{
			name:           "session was not found",
			giveReqVars:    map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			wantStatusCode: http.StatusNotFound,
			wantContent:    "session with UUID aa-bb-cc-dd was not found",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _  = http.NewRequest(http.MethodGet, "http://testing", http.NoBody)
				rr      = httptest.NewRecorder()
				stor    = storage.NewInMemoryStorage(time.Second*2, 32)
				ps      = pubsub.NewInMemory()
				handler = session.NewHandler(context.Background(), config.Config{}, stor, ps, ps, &fakeMetrics{})
			)

			defer func() { _ = stor.Close(); _ = ps.Close() }()

			if tt.giveReqVars != nil {
				req = mux.SetURLVars(req, tt.giveReqVars)
			}

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.wantContent)
		})
	}
}

func TestHandler_ServeHTTPSuccessSingle(t *testing.T) {
	t.Skip("Not implemented :(")
}
