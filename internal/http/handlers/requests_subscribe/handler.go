package requests_subscribe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath

	Handler struct {
		db       storage.Storage
		sub      pubsub.Subscriber[pubsub.RequestEvent]
		upgrader websocket.Upgrader
	}
)

func New(db storage.Storage, sub pubsub.Subscriber[pubsub.RequestEvent]) *Handler {
	return &Handler{db: db, sub: sub}
}

func (h *Handler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request, sID sID) error {
	if _, err := h.db.GetSession(ctx, sID.String()); err != nil {
		return fmt.Errorf("failed to get the session: %w", err)
	}

	// upgrade the connection to the WebSocket
	ws, upgErr := h.upgrader.Upgrade(w, r, http.Header{})
	if upgErr != nil {
		return fmt.Errorf("failed to upgrade the connection: %w", upgErr)
	}

	defer func() { _ = ws.Close() }()

	// create a new context for the request
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// uncomment to debug the ping/pong messages
	// ws.SetPongHandler(func(appData string) error { fmt.Println(">>> pong", appData); return nil })

	sub, unsubscribe, err := h.sub.Subscribe(ctx, sID.String())
	if err != nil {
		return fmt.Errorf("failed to subscribe to the captured requests for the session %s: %w", sID.String(), err)
	}

	defer unsubscribe()

	// read messages from the client in a separate goroutine and cancel the context when the connection is closed or
	// an error occurs
	go func() { defer cancel(); _ = h.reader(ctx, ws) }()

	// run a loop that sends routing updates to the client and pings the client periodically
	return h.writer(ctx, ws, sub)
}

// reader is a function that reads messages from the client. It must be run in a separate goroutine to prevent
// blocking. This function will exit when the context is canceled, the client closes the connection, or an error
// during the reading occurs.
func (*Handler) reader(ctx context.Context, ws *websocket.Conn) error {
	for {
		if ctx.Err() != nil { // check if the context is canceled
			return nil
		}

		var messageType, msgReader, msgErr = ws.NextReader() // TODO: is there any way to avoid locking without context?
		if msgErr != nil {
			return msgErr
		}

		if msgReader != nil {
			_, _ = io.Copy(io.Discard, msgReader) // ignore the message body but read it to prevent potential memory leaks
		}

		if messageType == websocket.CloseMessage {
			return nil // client closed the connection
		}
	}
}

// writer is a function that writes messages to the client. It may NOT be run in a separate goroutine because it
// will block until the context is canceled, the client closes the connection, or an error during the writing occurs.
//
// This function sends the captured requests to the client and pings the client periodically.
func (h *Handler) writer(ctx context.Context, ws *websocket.Conn, sub <-chan pubsub.RequestEvent) error { //nolint:funlen
	const pingInterval, pingDeadline = 10 * time.Second, 5 * time.Second

	// create a ticker for the ping messages
	var pingTicker = time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done(): // check if the context is canceled
			return nil

		case r, isOpened := <-sub: // wait for the captured requests
			if !isOpened {
				return nil // this should never happen, but just in case
			}

			var (
				action  openapi.RequestEventAction
				request *openapi.RequestEventRequest
			)

			switch r.Action {
			case pubsub.RequestActionCreate:
				action = openapi.RequestEventActionCreate
			case pubsub.RequestActionDelete:
				action = openapi.RequestEventActionDelete
			case pubsub.RequestActionClear:
				action = openapi.RequestEventActionClear
			default:
				continue // skip the unknown action
			}

			if r.Request != nil {
				rID, pErr := uuid.Parse(r.Request.ID)
				if pErr != nil {
					continue
				}

				var rHeaders = make([]openapi.HttpHeader, len(r.Request.Headers))
				for i, header := range r.Request.Headers {
					rHeaders[i].Name, rHeaders[i].Value = header.Name, header.Value
				}

				request = &openapi.RequestEventRequest{
					Uuid:                rID,
					CapturedAtUnixMilli: r.Request.CreatedAtUnixMilli,
					ClientAddress:       r.Request.ClientAddr,
					Headers:             rHeaders,
					Method:              strings.ToUpper(r.Request.Method),
					Url:                 r.Request.URL,
				}
			}

			// write the response to the client
			if err := ws.WriteJSON(openapi.RequestEvent{Action: action, Request: request}); err != nil {
				return fmt.Errorf("failed to write the message: %w", err)
			}

		case <-pingTicker.C: // send ping messages to the client
			if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(pingDeadline)); err != nil {
				return fmt.Errorf("failed to send the ping message: %w", err)
			}
		}
	}
}
