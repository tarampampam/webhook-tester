package session

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type Handler struct {
	ctx context.Context

	cfg  config.Config
	stor storage.Storage
	pub  pubsub.Publisher
	sub  pubsub.Subscriber

	connCounter int32 // atomic usage only!
	upgrader    websocket.Upgrader
	json        jsoniter.API
}

func NewHandler(
	ctx context.Context,
	cfg config.Config,
	stor storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
) *Handler {
	return &Handler{
		ctx:  ctx,
		cfg:  cfg,
		stor: stor,
		pub:  pub,
		sub:  sub,

		upgrader: websocket.Upgrader{
			ReadBufferSize:  512, //nolint:gomnd
			WriteBufferSize: 512, //nolint:gomnd
		},
		json: jsoniter.ConfigFastest,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if limit := h.cfg.WebSockets.MaxClients; limit != 0 {
		if atomic.LoadInt32(&h.connCounter) > int32(limit) {
			http.Error(w, "too many active connections", http.StatusTooManyRequests)

			return
		}
	}

	// extract session UUID from request path
	sessionUUID, exists := mux.Vars(r)["sessionUUID"]
	if !exists {
		http.Error(w, "cannot extract session UUID", http.StatusInternalServerError)

		return
	}

	// verify that session exists in a storage
	if session, err := h.stor.GetSession(sessionUUID); session == nil {
		if err != nil {
			http.Error(w, "cannot read session data: "+err.Error(), http.StatusInternalServerError)

			return
		}

		http.Error(w, "session with UUID "+sessionUUID+" was not found", http.StatusNotFound)

		return
	}

	// upgrade the HTTP server connection to the WebSocket protocol
	conn, upgradingErr := h.upgrader.Upgrade(w, r, nil)
	if upgradingErr != nil {
		return
	}

	atomic.AddInt32(&h.connCounter, 1) // increment active connections count

	go h.serveWebsocketConnection(sessionUUID, conn)
}

// newClientContext creates new context for the client (connection).
func (h *Handler) newClientContext() (context.Context, context.CancelFunc) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if lifetime := h.cfg.WebSockets.MaxLifetime; lifetime > time.Duration(0) {
		ctx, cancel = context.WithTimeout(h.ctx, lifetime)
	} else {
		ctx, cancel = context.WithCancel(h.ctx)
	}

	return ctx, cancel
}

const (
	pingInterval     = time.Second * 10 // pinging interval
	eventsBufferSize = 64               // buffer size for events subscription
)

// serveWebsocketConnection serves websocket connection.
func (h *Handler) serveWebsocketConnection(sessionUUID string, conn *websocket.Conn) {
	defer func() {
		_ = conn.Close() // close connection

		atomic.AddInt32(&h.connCounter, -1) // decrement active connections count
	}()

	// create channel for events (do NOT close him unless you are sure no one is writing into it)
	var eventsCh = make(chan pubsub.Event, eventsBufferSize)

	// subscribe to events
	if err := h.sub.Subscribe(sessionUUID, eventsCh); err != nil {
		return
	}

	// gracefully unsubscribing from events
	defer func() {
		_ = h.sub.Unsubscribe(sessionUUID, eventsCh)

		t := time.NewTicker(time.Microsecond * 3) //nolint:gomnd
		defer t.Stop()

		for { // cleanup the channel with a little intervals (a "little bit" dirty hack)
			select {
			case <-eventsCh:
				<-t.C

			default:
				// WARNING: channel closing may occurs the panic on the pub/sub implementation side (comment next line
				// in this case)
				close(eventsCh)

				return
			}
		}
	}()

	ctx, cancel := h.newClientContext()
	defer cancel()

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case event := <-eventsCh:
			j, _ := h.json.Marshal(output{Name: event.Name(), Date: string(event.Data())})

			if err := conn.WriteMessage(websocket.TextMessage, j); err != nil {
				return // cannot write into websocket (client has left the channel)
			}

		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return // client pinging using websocket has been failed
			}
		}
	}
}

type output struct {
	Name string `json:"name"`
	Date string `json:"data"`
}
