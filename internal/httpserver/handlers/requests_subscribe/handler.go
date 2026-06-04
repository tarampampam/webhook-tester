package requests_subscribe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
)

type sID = openapi.SessionUUIDInPath

const (
	pingInterval = 30 * time.Second
	pingDeadline = 5 * time.Second
)

// Handler handles WebSocket subscriptions for real-time captured request notifications.
type Handler struct {
	ps       pubsub.Subscriber
	upgrader websocket.Upgrader
}

// New creates a new Handler.
func New(ps pubsub.Subscriber) *Handler { return &Handler{ps: ps} }

// Handle upgrades the connection to WebSocket and streams request events for the given session.
func (h *Handler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request, sessionID sID) error {
	sub, unsubscribe, subErr := h.ps.Subscribe(ctx, sessionID)
	if subErr != nil {
		return fmt.Errorf("subscribe: %w", subErr)
	}

	defer unsubscribe()

	ws, upgErr := h.upgrader.Upgrade(w, r, nil)
	if upgErr != nil {
		return fmt.Errorf("upgrade: %w", upgErr)
	}

	defer func() { _ = ws.Close() }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() { defer cancel(); h.wsRead(ctx, ws) }()

	if err := h.wsWrite(ctx, ws, sub); err != nil {
		// connection is already upgraded - HTTP error response is no longer possible; log and exit cleanly
		logger.FromContext(ctx).Warn("websocket session closed with error", logger.Error(err))
	}

	return nil
}

// wsRead reads and discards incoming WebSocket messages. It returns when the connection closes or ctx is done.
func (*Handler) wsRead(ctx context.Context, ws *websocket.Conn) {
	for ctx.Err() == nil {
		_, r, err := ws.NextReader()
		if err != nil {
			return
		}

		if r != nil {
			if _, err = io.Copy(io.Discard, r); err != nil {
				return
			}
		}
	}
}

// wsWrite sends pubsub events to the WebSocket client and pings the client periodically.
func (*Handler) wsWrite(ctx context.Context, ws *websocket.Conn, sub <-chan pubsub.RequestEvent) error {
	ping := time.NewTicker(pingInterval)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-sub:
			if !ok {
				return nil
			}

			msg, skip := toOpenAPIEvent(event)
			if skip {
				continue
			}

			if err := ws.WriteJSON(msg); err != nil {
				return fmt.Errorf("write: %w", err)
			}

		case <-ping.C:
			if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(pingDeadline)); err != nil {
				return fmt.Errorf("ping: %w", err)
			}
		}
	}
}

// toOpenAPIEvent converts a pubsub event to the OpenAPI wire format.
// Returns skip=true for unknown actions.
func toOpenAPIEvent(e pubsub.RequestEvent) (openapi.RequestEvent, bool /* skip */) {
	var action openapi.RequestEventAction

	switch e.Action {
	case pubsub.RequestActionCreate:
		action = openapi.RequestEventActionCreate
	case pubsub.RequestActionDelete:
		action = openapi.RequestEventActionDelete
	case pubsub.RequestActionClear:
		action = openapi.RequestEventActionClear
	default:
		return openapi.RequestEvent{}, true
	}

	msg := openapi.RequestEvent{Action: action}

	if e.Request != nil {
		headers := make([]openapi.HttpHeader, len(e.Request.Headers))
		for i, hdr := range e.Request.Headers {
			headers[i] = openapi.HttpHeader{Name: hdr.Name, Value: hdr.Value}
		}

		msg.Request = &openapi.RequestEventRequest{
			Uuid:                e.Request.ID,
			CapturedAtUnixMilli: e.Request.CreatedAt.UnixMilli(),
			ClientAddress:       e.Request.ClientAddr,
			Headers:             headers,
			Method:              e.Request.Method,
			Url:                 e.Request.URL,
		}
	}

	return msg, false
}
