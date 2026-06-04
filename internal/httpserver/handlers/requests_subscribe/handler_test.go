package requests_subscribe_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/requests_subscribe"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

const fixedSID = "550e8400-e29b-41d4-a716-446655440000"

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("subscribe error returned before websocket upgrade", func(t *testing.T) {
		t.Parallel()

		h := requests_subscribe.New(&mockSubscriber{
			subscribeFn: func(_ context.Context, _ string) (<-chan pubsub.RequestEvent, func(), error) {
				return nil, func() {}, errors.New("pubsub unavailable")
			},
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		err := h.Handle(t.Context(), w, r, fixedSID)

		assert.ErrorContains(t, err, "subscribe")
		assert.ErrorContains(t, err, "pubsub unavailable")
		assert.True(t, w.Code != http.StatusSwitchingProtocols)
	})

	t.Run("create event with full request data forwarded over websocket", func(t *testing.T) {
		t.Parallel()

		fixedTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

		ch, conn := dialWS(t)

		ch <- pubsub.RequestEvent{
			Action: pubsub.RequestActionCreate,
			Request: &pubsub.RequestData{
				ID:         "req-uuid-1",
				ClientAddr: "1.2.3.4",
				Method:     "POST",
				Headers:    []pubsub.HttpHeader{{Name: "Content-Type", Value: "application/json"}},
				URL:        "https://example.com/hook",
				CreatedAt:  fixedTime,
			},
		}

		var got openapi.RequestEvent

		assert.NoError(t, conn.ReadJSON(&got))
		assert.Equal(t, openapi.RequestEventActionCreate, got.Action)
		assert.NotNil(t, got.Request)
		assert.Equal(t, "req-uuid-1", got.Request.Uuid)
		assert.Equal(t, fixedTime.UnixMilli(), got.Request.CapturedAtUnixMilli)
		assert.Equal(t, "1.2.3.4", got.Request.ClientAddress)
		assert.Equal(t, "POST", got.Request.Method)
		assert.Equal(t, "https://example.com/hook", got.Request.Url)
		assert.DeepEqual(t, []openapi.HttpHeader{{Name: "Content-Type", Value: "application/json"}}, got.Request.Headers)
	})

	t.Run("delete event forwarded over websocket", func(t *testing.T) {
		t.Parallel()

		ch, conn := dialWS(t)

		ch <- pubsub.RequestEvent{
			Action:  pubsub.RequestActionDelete,
			Request: &pubsub.RequestData{ID: "req-uuid-2"},
		}

		var got openapi.RequestEvent

		assert.NoError(t, conn.ReadJSON(&got))
		assert.Equal(t, openapi.RequestEventActionDelete, got.Action)
		assert.NotNil(t, got.Request)
		assert.Equal(t, "req-uuid-2", got.Request.Uuid)
	})

	t.Run("clear event forwarded with nil request field", func(t *testing.T) {
		t.Parallel()

		ch, conn := dialWS(t)

		ch <- pubsub.RequestEvent{Action: pubsub.RequestActionClear}

		var got openapi.RequestEvent

		assert.NoError(t, conn.ReadJSON(&got))
		assert.Equal(t, openapi.RequestEventActionClear, got.Action)
		assert.Nil(t, got.Request)
	})

	t.Run("unknown action is not forwarded", func(t *testing.T) {
		t.Parallel()

		ch, conn := dialWS(t)

		ch <- pubsub.RequestEvent{Action: "bogus"}

		ch <- pubsub.RequestEvent{Action: pubsub.RequestActionClear}

		var got openapi.RequestEvent

		assert.NoError(t, conn.ReadJSON(&got))
		assert.Equal(t, openapi.RequestEventActionClear, got.Action)
	})

	t.Run("incoming client messages are discarded and handler keeps running", func(t *testing.T) {
		t.Parallel()

		ch, conn := dialWS(t)

		assert.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("ignored")))

		ch <- pubsub.RequestEvent{Action: pubsub.RequestActionClear}

		var got openapi.RequestEvent

		assert.NoError(t, conn.ReadJSON(&got))
		assert.Equal(t, openapi.RequestEventActionClear, got.Action)
	})

	t.Run("handler exits when pubsub channel is closed", func(t *testing.T) {
		t.Parallel()

		ch, conn := dialWS(t)

		close(ch)

		assert.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

		_, _, err := conn.ReadMessage()

		assert.Error(t, err)
	})

	t.Run("handler exits when client disconnects", func(t *testing.T) {
		t.Parallel()

		_, conn := dialWS(t)

		assert.NoError(t, conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		))

		assert.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))

		_, _, err := conn.ReadMessage()

		assert.Error(t, err)
	})
}

// --------------------------------------------------------------------------------------------------------------------

// dialWS starts the handler inside a httptest.Server and returns a connected WebSocket client
// plus the event channel wired into the handler. The subscription topic is asserted to equal fixedSID.
func dialWS(t *testing.T) (chan pubsub.RequestEvent, *websocket.Conn) {
	t.Helper()

	ch := make(chan pubsub.RequestEvent, 16)

	h := requests_subscribe.New(&mockSubscriber{
		subscribeFn: func(_ context.Context, topic string) (<-chan pubsub.RequestEvent, func(), error) {
			assert.Equal(t, fixedSID, topic)

			return ch, func() {}, nil
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, h.Handle(r.Context(), w, r, fixedSID))
	}))

	t.Cleanup(srv.Close)

	conn, resp, err := (&websocket.Dialer{}).Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	if resp != nil && resp.Body != nil {
		assert.NoError(t, resp.Body.Close())
	}

	t.Cleanup(func() { assert.NoError(t, conn.Close()) })

	return ch, conn
}

type mockSubscriber struct {
	subscribeFn func(context.Context, string) (<-chan pubsub.RequestEvent, func(), error)
}

var _ pubsub.Subscriber = (*mockSubscriber)(nil)

func (m *mockSubscriber) Subscribe(ctx context.Context, topic string) (<-chan pubsub.RequestEvent, func(), error) {
	if m.subscribeFn == nil {
		panic("mockSubscriber: unexpected Subscribe call")
	}

	return m.subscribeFn(ctx, topic)
}
