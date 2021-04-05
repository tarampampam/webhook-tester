package session

import (
	"context"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"

	"github.com/gorilla/mux"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Handler struct {
	ctx context.Context

	cfg  config.Config
	stor storage.Storage
	pub  pubsub.Publisher
	sub  pubsub.Subscriber
	log  *zap.Logger

	connCounter int32 // atomic usage only!
	upgrader    websocket.Upgrader
}

func NewHandler(
	ctx context.Context,
	cfg config.Config,
	stor storage.Storage,
	pub pubsub.Publisher,
	sub pubsub.Subscriber,
	log *zap.Logger, // TODO remove logger from this handler?
) *Handler {
	return &Handler{
		ctx:  ctx,
		cfg:  cfg,
		stor: stor,
		pub:  pub,
		sub:  sub,
		log:  log,

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,                                       //nolint:gomnd
			WriteBufferSize: 1024,                                       //nolint:gomnd
			CheckOrigin:     func(r *http.Request) bool { return true }, // FIXME remove this, just for a tests
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { //nolint:funlen,gocognit,gocyclo // TODO rewrite
	if limit := h.cfg.WebSockets.MaxClients; limit != 0 {
		if atomic.LoadInt32(&h.connCounter) >= int32(limit) {
			http.Error(w, "too many active connections", http.StatusTooManyRequests)

			return
		}
	}

	// extract session UUID from request path
	sessionUUID, ok := mux.Vars(r)["sessionUUID"]
	if !ok {
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

	h.log.Debug("websocket connection established",
		zap.String("session UUID", sessionUUID),
		zap.String("local addr", conn.LocalAddr().String()),
		zap.String("remote addr", conn.RemoteAddr().String()),
		zap.Int32("current connections count", atomic.LoadInt32(&h.connCounter)),
	)

	go func() {
		defer func() { atomic.AddInt32(&h.connCounter, -1) }() // decrement active connections count

		var (
			ctx    context.Context
			cancel context.CancelFunc
		)

		if lifetime := h.cfg.WebSockets.MaxLifetime; lifetime > time.Duration(0) {
			ctx, cancel = context.WithTimeout(h.ctx, lifetime)
		} else {
			ctx, cancel = context.WithCancel(h.ctx)
		}

		defer cancel()

		pingTicker := time.NewTicker(time.Second * 10) //nolint:gomnd
		defer pingTicker.Stop()

		eventsCh := make(chan pubsub.Event, 32) //nolint:gomnd // TODO what is the better chan size?

		if err := h.sub.Subscribe(sessionUUID, eventsCh); err != nil {
			h.log.Error("cannot subscribe to pub/sub", zap.Error(err))

			return
		}

		defer func() { // gracefully unsubscribing from the events
			_ = h.sub.Unsubscribe(sessionUUID, eventsCh)

			for { // cleanup the channel before closing
				select {
				case <-eventsCh:
					runtime.Gosched()

				default:
					return
				}
			}
		}()

		defer func() {
			if closingErr := conn.Close(); closingErr != nil {
				h.log.Error("websocket closing failed",
					zap.Error(closingErr),
					zap.String("local addr", conn.LocalAddr().String()),
					zap.String("remote addr", conn.RemoteAddr().String()),
				)
			}
		}()

		go func() { // TODO just for a test
			for {
				select {
				case <-ctx.Done():
					return

				case <-time.After(time.Second):
					if pubErr := h.pub.Publish(sessionUUID, pubsub.NewRequestRegisteredEvent("foo request UUID")); pubErr != nil {
						h.log.Error("cannot publish event", zap.Error(pubErr))
					}
				}
			}
		}()

	loop:
		for {
			select {
			case <-h.ctx.Done():
				break loop

			case event, opened := <-eventsCh:
				if !opened {
					break loop
				}

				if err := conn.WriteMessage(websocket.TextMessage, event.Data()); err != nil {
					h.log.Debug("cannot write into websocket", zap.Error(err))

					break loop
				}

			case <-pingTicker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil { // client pinging failed
					h.log.Debug("client pinging using websocket has been failed", zap.Error(err))

					break loop
				}
			}
		}
	}()
}
