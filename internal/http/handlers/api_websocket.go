package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type websocketMetrics interface {
	IncrementActiveClients()
	DecrementActiveClients()
}

type apiWebsocket struct {
	ctx     context.Context
	cfg     config.Config
	stor    storage.Storage
	pub     pubsub.Publisher
	sub     pubsub.Subscriber
	metrics websocketMetrics

	connCounter atomic.Int32
}

var upgrader = websocket.Upgrader{ //nolint:gochecknoglobals
	ReadBufferSize:  512, //nolint:gomnd
	WriteBufferSize: 512, //nolint:gomnd
}

const (
	pingInterval     = time.Second * 10 // pinging interval
	eventsBufferSize = 64               // buffer size for events subscription
)

// WebsocketSession returns websocket session.
func (s *apiWebsocket) WebsocketSession(c echo.Context, sessionUuid api.SessionUUID) error { //nolint:funlen
	// is the limit exceeded?
	if limit := s.cfg.WebSockets.MaxClients; limit != 0 {
		if s.connCounter.Load() > int32(limit) {
			return c.JSON(http.StatusTooManyRequests, api.NotFound{
				Code:    http.StatusTooManyRequests,
				Message: "Too many active connections",
			})
		}
	}

	// verify that session exists in a storage
	if session, err := s.stor.GetSession(sessionUuid.String()); session == nil {
		if err != nil {
			return c.JSON(http.StatusInternalServerError, api.ServerError{
				Code:    http.StatusInternalServerError,
				Message: errors.Wrap(err, "cannot read session data").Error(),
			})
		}

		return c.JSON(http.StatusNotFound, api.ServerError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("the session with UUID %s was not found", sessionUuid.String()),
		})
	}

	// upgrade the HTTP server connection to the WebSocket protocol
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), http.Header{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	}

	defer func() { _ = ws.Close() }()

	s.connCounter.Add(1) // increment active connections count
	s.metrics.IncrementActiveClients()

	defer func() {
		s.connCounter.Add(-1)
		s.metrics.DecrementActiveClients()
	}()

	// create channel for events (do NOT close it unless you are sure no one is writing into it)
	var eventsCh = make(chan pubsub.Event, eventsBufferSize)

	// subscribe to events
	if err = s.sub.Subscribe(sessionUuid.String(), eventsCh); err != nil {
		return c.JSON(http.StatusInternalServerError, api.ServerError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	}

	// gracefully unsubscribe from events
	defer func() { _ = s.sub.Unsubscribe(sessionUuid.String(), eventsCh) }()

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if lifetime := s.cfg.WebSockets.MaxLifetime; lifetime > time.Duration(0) {
		ctx, cancel = context.WithTimeout(s.ctx, lifetime)
	} else {
		ctx, cancel = context.WithCancel(s.ctx)
	}

	defer cancel()

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for { // TODO run in a goroutine?
		select {
		case <-ctx.Done():
			return c.NoContent(http.StatusGone)

		case event := <-eventsCh:
			j, _ := json.Marshal(api.WebsocketPayload{
				Name: api.WebsocketPayloadName(event.Name()),
				Data: string(event.Data()),
			})

			if err = ws.WriteMessage(websocket.TextMessage, j); err != nil {
				return c.NoContent(http.StatusGone) // cannot write into websocket (client has left the channel)
			}

		case <-pingTicker.C:
			if err = ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return c.NoContent(http.StatusUnprocessableEntity) // client pinging using websocket has been failed
			}
		}
	}
}
