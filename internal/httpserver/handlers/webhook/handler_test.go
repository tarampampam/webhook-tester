package webhook_test

import (
	"context"
	"errors"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/webhook"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

const testSID = "550e8400-e29b-41d4-a716-446655440000"

func newNopPublisher() *mockPublisher {
	return &mockPublisher{publishFn: func(context.Context, string, pubsub.RequestEvent) error { return nil }}
}

func TestShouldBeCaptured(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		givePath string
		want     bool
	}{
		"uuid only":             {givePath: "/" + testSID, want: true},
		"uuid trailing slash":   {givePath: "/" + testSID + "/", want: true},
		"uuid with sub-path":    {givePath: "/" + testSID + "/200", want: true},
		"uuid uppercase hex":    {givePath: "/" + strings.ToUpper(testSID), want: true},
		"empty path":            {givePath: "/", want: false},
		"non-uuid path":         {givePath: "/api/sessions", want: false},
		"uuid too short":        {givePath: "/" + testSID[:35], want: false},
		"no dashes in uuid":     {givePath: "/" + strings.ReplaceAll(testSID, "-", "0"), want: false},
		"invalid chars in uuid": {givePath: "/" + testSID[:32] + "zzzz", want: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, tt.givePath, nil)

			assert.Equal(t, tt.want, webhook.ShouldBeCaptured(r))
		})
	}
}

func TestHandler_Handle(t *testing.T) { //nolint:funlen
	t.Parallel()

	okMeta := &storage.RequestMeta{CreatedAt: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)}

	for name, tt := range map[string]struct {
		givePath       string
		giveMethod     string
		giveBodyStr    string
		giveMaxBody    uint
		mockGetSession func(context.Context, string) (*storage.Session, error)
		mockAddTTL     func(context.Context, string, time.Duration) error
		mockNewRequest func(context.Context, string, string, storage.CapturedRequest) (*storage.RequestMeta, error)
		wantStatus     int
		checkResp      func(*testing.T, *httptest.ResponseRecorder)
	}{
		"invalid session id in url": {
			givePath:   "/not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},

		"session not found auto-create disabled": {
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return nil, storage.ErrSessionNotFound
			},
			wantStatus: http.StatusNotFound,
		},

		"session storage error": {
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return nil, errors.New("storage unavailable")
			},
			wantStatus: http.StatusInternalServerError,
		},

		"session with expiry add-ttl fails": {
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{Code: http.StatusOK},
					Meta:     storage.SessionMeta{ExpiresAt: time.Now().Add(time.Hour)},
				}, nil
			},
			mockAddTTL: func(_ context.Context, _ string, _ time.Duration) error {
				return errors.New("ttl failure")
			},
			wantStatus: http.StatusInternalServerError,
		},

		"session with expiry ttl is refreshed": {
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{Code: http.StatusOK},
					Meta:     storage.SessionMeta{ExpiresAt: time.Now().Add(time.Hour)},
				}, nil
			},
			mockAddTTL: func(_ context.Context, sID string, howMuch time.Duration) error {
				assert.Equal(t, testSID, sID)
				assert.Equal(t, time.Hour, howMuch)

				return nil
			},
			mockNewRequest: func(_ context.Context, _, _ string, _ storage.CapturedRequest) (*storage.RequestMeta, error) {
				return okMeta, nil
			},
			wantStatus: http.StatusOK,
		},

		"body too large": {
			giveMaxBody: 10,
			giveBodyStr: strings.Repeat("x", 11),
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
			},
			wantStatus: http.StatusRequestEntityTooLarge,
		},

		"unlimited body size (0) accepts large body": {
			giveMaxBody: 0,
			giveBodyStr: strings.Repeat("x", 10_000),
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
			},
			mockNewRequest: func(_ context.Context, _, _ string, req storage.CapturedRequest) (*storage.RequestMeta, error) {
				assert.Equal(t, 10_000, len(req.Body))

				return okMeta, nil
			},
			wantStatus: http.StatusOK,
		},

		"new request storage fails": {
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
			},
			mockNewRequest: func(_ context.Context, _, _ string, _ storage.CapturedRequest) (*storage.RequestMeta, error) {
				return nil, errors.New("disk full")
			},
			wantStatus: http.StatusInternalServerError,
		},

		"successful capture echoes session response": {
			giveMethod:  http.MethodPut,
			giveBodyStr: "payload",
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{
						Code:    418,
						Body:    []byte("hello"),
						Headers: []storage.ResponseHeader{{Name: "X-Custom", Value: "custom-val"}},
					},
				}, nil
			},
			mockNewRequest: func(_ context.Context, sID, _ string, req storage.CapturedRequest) (*storage.RequestMeta, error) {
				assert.Equal(t, testSID, sID)
				assert.Equal(t, http.MethodPut, req.Method)
				assert.Equal(t, "payload", string(req.Body))

				return okMeta, nil
			},
			wantStatus: 418,
			checkResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "hello", w.Body.String())
				assert.Equal(t, "5", w.Header().Get("Content-Length"))
				assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, 36, len(w.Header().Get("X-Wh-Request-Id")))
				assert.Equal(t, "custom-val", w.Header().Get("X-Custom"))
			},
		},

		"url path overrides response status code": {
			givePath: "/" + testSID + "/202",
			mockGetSession: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
			},
			mockNewRequest: func(_ context.Context, _, _ string, _ storage.CapturedRequest) (*storage.RequestMeta, error) {
				return okMeta, nil
			},
			wantStatus: http.StatusAccepted,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := &mockStorage{
				getSessionFn:    tt.mockGetSession,
				addSessionTTLFn: tt.mockAddTTL,
				newRequestFn:    tt.mockNewRequest,
			}

			h := webhook.New(logger.NewNop(), s, newNopPublisher(), false, time.Hour, tt.giveMaxBody)

			method := tt.giveMethod
			if method == "" {
				method = http.MethodPost
			}

			path := tt.givePath
			if path == "" {
				path = "/" + testSID
			}

			var body io.Reader
			if tt.giveBodyStr != "" {
				body = strings.NewReader(tt.giveBodyStr)
			}

			r := httptest.NewRequest(method, path, body)
			w := httptest.NewRecorder()

			h.Handle(w, r)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.checkResp != nil {
				tt.checkResp(t, w)
			}
		})
	}
}

func TestHandler_Handle_AutoCreate(t *testing.T) {
	t.Parallel()

	t.Run("session created automatically with x-wh-created-automatically header", func(t *testing.T) {
		t.Parallel()

		var getSessionCalls int

		s := &mockStorage{
			getSessionFn: func(_ context.Context, sID string) (*storage.Session, error) {
				getSessionCalls++
				if getSessionCalls == 1 {
					return nil, storage.ErrSessionNotFound
				}

				assert.Equal(t, testSID, sID)

				return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
			},
			newSessionFn: func(_ context.Context, sID string, resp storage.SessionResponse, ttl time.Duration) (*storage.SessionMeta, error) {
				assert.Equal(t, testSID, sID)
				assert.Equal(t, uint16(http.StatusOK), resp.Code)
				assert.Equal(t, time.Hour, ttl)

				return &storage.SessionMeta{}, nil
			},
			newRequestFn: func(_ context.Context, _, _ string, _ storage.CapturedRequest) (*storage.RequestMeta, error) {
				return &storage.RequestMeta{CreatedAt: time.Now()}, nil
			},
		}

		h := webhook.New(logger.NewNop(), s, newNopPublisher(), true, time.Hour, 0)

		r := httptest.NewRequest(http.MethodPost, "/"+testSID, nil)
		w := httptest.NewRecorder()

		h.Handle(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "1", w.Header().Get("X-Wh-Created-Automatically"))
	})

	t.Run("new session creation fails returns 500", func(t *testing.T) {
		t.Parallel()

		s := &mockStorage{
			getSessionFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return nil, storage.ErrSessionNotFound
			},
			newSessionFn: func(_ context.Context, _ string, _ storage.SessionResponse, _ time.Duration) (*storage.SessionMeta, error) {
				return nil, errors.New("storage write error")
			},
		}

		h := webhook.New(logger.NewNop(), s, newNopPublisher(), true, time.Hour, 0)

		r := httptest.NewRequest(http.MethodPost, "/"+testSID, nil)
		w := httptest.NewRecorder()

		h.Handle(w, r)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandler_Handle_ResponseDelay(t *testing.T) {
	t.Parallel()

	const delay = 5 * time.Millisecond

	s := &mockStorage{
		getSessionFn: func(_ context.Context, _ string) (*storage.Session, error) {
			return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK, Delay: delay}}, nil
		},
		newRequestFn: func(_ context.Context, _, _ string, _ storage.CapturedRequest) (*storage.RequestMeta, error) {
			return &storage.RequestMeta{CreatedAt: time.Now()}, nil
		},
	}

	h := webhook.New(logger.NewNop(), s, newNopPublisher(), false, time.Hour, 0)

	r := httptest.NewRequest(http.MethodPost, "/"+testSID, nil)
	w := httptest.NewRecorder()

	start := time.Now()

	h.Handle(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, time.Since(start) >= delay)
}

func TestHandler_Handle_PubSubEvent(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	publishCh := make(chan pubsub.RequestEvent, 1)

	s := &mockStorage{
		getSessionFn: func(_ context.Context, _ string) (*storage.Session, error) {
			return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
		},
		newRequestFn: func(_ context.Context, _, _ string, _ storage.CapturedRequest) (*storage.RequestMeta, error) {
			return &storage.RequestMeta{CreatedAt: fixedTime}, nil
		},
	}

	ps := &mockPublisher{
		publishFn: func(_ context.Context, topic string, ev pubsub.RequestEvent) error {
			assert.Equal(t, testSID, topic)

			publishCh <- ev

			return nil
		},
	}

	h := webhook.New(logger.NewNop(), s, ps, false, time.Hour, 0)

	r := httptest.NewRequest(http.MethodPost, "/"+testSID, strings.NewReader("body"))
	w := httptest.NewRecorder()

	h.Handle(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case ev := <-publishCh:
		assert.Equal(t, pubsub.RequestActionCreate, ev.Action)
		assert.NotNil(t, ev.Request)
		assert.Equal(t, http.MethodPost, ev.Request.Method)
		assert.Equal(t, fixedTime, ev.Request.CreatedAt)
	case <-time.After(time.Second):
		t.Fatal("pubsub publish goroutine did not fire within 1s")
	}
}

// TestHandler_Handle_PubSub_EventDeliveredToSlowSubscriber verifies that a published event reaches a
// subscriber that is not immediately ready to read - i.e. it starts consuming the channel only after
// the handler's background goroutine has already exited.
//
// The regression being tested: if the handler wraps the Publish context in WithTimeout + defer cancel(),
// cancel() fires the moment the goroutine exits (right after Publish returns). Delivery goroutines
// spawned by Memory.Publish then see ctx.Done() immediately ready, and since the subscriber hasn't
// read yet, they exit via that case and silently drop the event.
func TestHandler_Handle_PubSub_EventDeliveredToSlowSubscriber(t *testing.T) {
	t.Parallel()

	ps := pubsub.NewMemory()

	t.Cleanup(func() { _ = ps.Close() })

	sub, unsub, err := ps.Subscribe(t.Context(), testSID)
	assert.NoError(t, err)

	defer unsub()

	// publishReturned signals the moment Memory.Publish has returned inside the goroutine.
	// right after this, the goroutine exits - triggering defer cancel() in the buggy version.
	var publishReturned sync.WaitGroup

	publishReturned.Add(1)

	pub := &publishBarrier{Publisher: ps, done: &publishReturned}

	s := &mockStorage{
		getSessionFn: func(_ context.Context, _ string) (*storage.Session, error) {
			return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
		},
		newRequestFn: func(_ context.Context, _, _ string, _ storage.CapturedRequest) (*storage.RequestMeta, error) {
			return &storage.RequestMeta{CreatedAt: time.Now()}, nil
		},
	}

	h := webhook.New(logger.NewNop(), s, pub, false, time.Hour, 0)

	r := httptest.NewRequest(http.MethodPost, "/"+testSID, nil)
	w := httptest.NewRecorder()
	h.Handle(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	// wait until Publish has returned, then sleep long enough for the goroutine to exit
	// and defer cancel() to fire - so delivery goroutines see a closed ctx.Done() before
	// the subscriber reads.
	const afterPublishDelay = 20 * time.Millisecond

	publishReturned.Wait()
	time.Sleep(afterPublishDelay)

	// subscriber reads after the goroutine has exited.
	// with the fix: ctx.Done() is nil; delivery goroutine is still blocking on the channel
	//               and completes the rendezvous here.
	// with the bug: ctx.Done() fired during the sleep; delivery goroutine exited without
	//               delivering; the channel is empty and the select times out.
	select {
	case ev := <-sub:
		assert.Equal(t, pubsub.RequestActionCreate, ev.Action)
	case <-time.After(time.Second):
		t.Fatal("event not delivered: publish context was canceled before subscriber could read")
	}
}

// --------------------------------------------------------------------------------------------------------------------

// mockStorage implements [storage.Storage] for handler isolation tests.
type mockStorage struct {
	getSessionFn    func(context.Context, string) (*storage.Session, error)
	newSessionFn    func(context.Context, string, storage.SessionResponse, time.Duration) (*storage.SessionMeta, error)
	addSessionTTLFn func(context.Context, string, time.Duration) error
	deleteSessionFn func(context.Context, string) error
	newRequestFn    func(context.Context, string, string, storage.CapturedRequest) (*storage.RequestMeta, error)
}

var _ storage.Storage = (*mockStorage)(nil)

func (m *mockStorage) GetSession(ctx context.Context, sID string) (*storage.Session, error) {
	if m.getSessionFn == nil {
		panic("mockStorage: unexpected GetSession call")
	}

	return m.getSessionFn(ctx, sID)
}

func (m *mockStorage) NewSession(
	ctx context.Context, sID string, resp storage.SessionResponse, ttl time.Duration,
) (*storage.SessionMeta, error) {
	if m.newSessionFn == nil {
		panic("mockStorage: unexpected NewSession call")
	}

	return m.newSessionFn(ctx, sID, resp, ttl)
}

func (m *mockStorage) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	if m.addSessionTTLFn == nil {
		panic("mockStorage: unexpected AddSessionTTL call")
	}

	return m.addSessionTTLFn(ctx, sID, howMuch)
}

func (m *mockStorage) DeleteSession(ctx context.Context, sID string) error {
	if m.deleteSessionFn == nil {
		panic("mockStorage: unexpected DeleteSession call")
	}

	return m.deleteSessionFn(ctx, sID)
}

func (m *mockStorage) NewRequest(
	ctx context.Context, sID, rID string, req storage.CapturedRequest,
) (*storage.RequestMeta, error) {
	if m.newRequestFn == nil {
		panic("mockStorage: unexpected NewRequest call")
	}

	return m.newRequestFn(ctx, sID, rID, req)
}

func (m *mockStorage) GetRequest(context.Context, string, string) (*storage.Request, error) {
	panic("mockStorage: GetRequest not expected in this handler")
}

func (m *mockStorage) GetRequests(context.Context, string, *error) (iter.Seq2[string, storage.Request], error) {
	panic("mockStorage: GetRequests not expected in this handler")
}

func (m *mockStorage) DeleteRequest(context.Context, string, string) error {
	panic("mockStorage: DeleteRequest not expected in this handler")
}

func (m *mockStorage) DeleteAllRequests(context.Context, string) error {
	panic("mockStorage: DeleteAllRequests not expected in this handler")
}

// mockPublisher implements [pubsub.Publisher] for handler isolation tests.
type mockPublisher struct {
	publishFn func(context.Context, string, pubsub.RequestEvent) error
}

var _ pubsub.Publisher = (*mockPublisher)(nil)

func (m *mockPublisher) Publish(ctx context.Context, topic string, ev pubsub.RequestEvent) error {
	if m.publishFn == nil {
		panic("mockPublisher: unexpected Publish call")
	}

	return m.publishFn(ctx, topic, ev)
}

// publishBarrier wraps a Publisher and signals done when Publish returns,
// letting tests synchronize on the exact moment the background goroutine
// exits (and defer cancel() fires in the buggy code).
type publishBarrier struct {
	pubsub.Publisher

	done *sync.WaitGroup
	once sync.Once
}

func (p *publishBarrier) Publish(ctx context.Context, topic string, ev pubsub.RequestEvent) error {
	err := p.Publisher.Publish(ctx, topic, ev)
	p.once.Do(p.done.Done)

	return err
}
