package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"gh.tarampamp.am/webhook-tester/v2/internal/config"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

// MockStorage is a mock type for the storage.Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) NewSession(ctx context.Context, session storage.Session, id ...string) (string, error) {
	args := m.Called(ctx, session, id)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetSession(ctx context.Context, sID string) (*storage.Session, error) {
	args := m.Called(ctx, sID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Session), args.Error(1)
}

func (m *MockStorage) AddSessionTTL(ctx context.Context, sID string, howMuch time.Duration) error {
	args := m.Called(ctx, sID, howMuch)
	return args.Error(0)
}

func (m *MockStorage) DeleteSession(ctx context.Context, sID string) error {
	args := m.Called(ctx, sID)
	return args.Error(0)
}

func (m *MockStorage) NewRequest(ctx context.Context, sID string, req storage.Request) (string, error) {
	args := m.Called(ctx, sID, req)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetRequest(ctx context.Context, sID, rID string) (*storage.Request, error) {
	args := m.Called(ctx, sID, rID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Request), args.Error(1)
}

func (m *MockStorage) GetAllRequests(ctx context.Context, sID string) (map[string]storage.Request, error) {
	args := m.Called(ctx, sID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]storage.Request), args.Error(1)
}

func (m *MockStorage) DeleteRequest(ctx context.Context, sID, rID string) error {
	args := m.Called(ctx, sID, rID)
	return args.Error(0)
}

func (m *MockStorage) DeleteAllRequests(ctx context.Context, sID string) error {
	args := m.Called(ctx, sID)
	return args.Error(0)
}

// MockPubSub is a mock for pubsub.Publisher
type MockPubSub struct {
	mock.Mock
}

func (m *MockPubSub) Publish(ctx context.Context, channelID string, event pubsub.RequestEvent) error {
	args := m.Called(ctx, channelID, event)
	return args.Error(0)
}

func (m *MockPubSub) Subscribe(ctx context.Context, channelID string) (<-chan pubsub.RequestEvent, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).(<-chan pubsub.RequestEvent), args.Error(1)
}

func (m *MockPubSub) Unsubscribe(ctx context.Context, channelID string, eventsCh <-chan pubsub.RequestEvent) error {
	args := m.Called(ctx, channelID, eventsCh)
	return args.Error(0)
}

func TestWebhookMiddleware_ProxyingEnabled_AppResponseMode(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	mockDB := new(MockStorage)
	mockPublisher := new(MockPubSub)
	appCfg := &config.AppSettings{
		SessionTTL:         time.Hour,
		MaxRequestBodySize: 1024,
	}

	// --- Mock Proxy Server 1 ---
	proxyServer1HitCount := 0
	proxyServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyServer1HitCount++
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/proxy1", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-value", r.Header.Get("X-Test-Header"))
		bodyBytes, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"key":"value"}`, string(bodyBytes))

		w.Header().Set("X-Proxy-Resp", "proxy1-ok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"proxy_ok":1}`))
	}))
	defer proxyServer1.Close()

	// --- Mock Proxy Server 2 ---
	proxyServer2HitCount := 0
	proxyServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyServer2HitCount++
		assert.Equal(t, "POST", r.Method)
		// Add more assertions for request to proxy2 if needed
		w.Header().Set("X-Proxy-Resp", "proxy2-ok")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"proxy_ok":2}`))
	}))
	defer proxyServer2.Close()

	sessionID := "test-session-proxy-app"
	requestBody := `{"key":"value"}`
	appResponseBody := `{"app_says":"hello"}`

	mockSession := &storage.Session{
		UUID:         sessionID,
		Code:         http.StatusOK,
		ResponseBody: []byte(appResponseBody),
		Headers:      []storage.HttpHeader{{Name: "Content-Type", Value: "application/json"}},
		ProxyURLs:    []string{proxyServer1.URL + "/proxy1", proxyServer2.URL + "/proxy2"},
		ProxyResponseMode: "app_response",
		CreatedAtUnixMilli: time.Now().UnixMilli(),
	}

	mockDB.On("GetSession", mock.Anything, sessionID).Return(mockSession, nil)
	mockDB.On("AddSessionTTL", mock.Anything, sessionID, mock.Anything).Return(nil)

	var capturedRequest storage.Request
	mockDB.On("NewRequest", mock.Anything, sessionID, mock.AnythingOfType("storage.Request")).Run(func(args mock.Arguments) {
		capturedRequest = args.Get(2).(storage.Request)
	}).Return("test-req-id", nil)

	// Mock GetRequest for the async publisher goroutine
	mockDB.On("GetRequest", mock.Anything, sessionID, "test-req-id").Return(&storage.Request{
		UUID: "test-req-id", // ensure some minimal data for publisher
		CreatedAtUnixMilli: time.Now().UnixMilli(),
	}, nil)
	mockPublisher.On("Publish", mock.Anything, sessionID, mock.AnythingOfType("pubsub.RequestEvent")).Return(nil)

	middleware := New(context.Background(), logger, mockDB, mockPublisher, appCfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is the next handler, which should not be hit if proxy logic takes over response
		// In app_response mode, this *is* hit.
	}))

	req := httptest.NewRequest("POST", "/"+sessionID+"/some/path", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Header", "test-value")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Response code should be from app's configured response")
	assert.Equal(t, appResponseBody, rr.Body.String(), "Response body should be from app's configured response")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))


	mockDB.AssertCalled(t, "GetSession", mock.Anything, sessionID)
	mockDB.AssertCalled(t, "NewRequest", mock.Anything, sessionID, mock.AnythingOfType("storage.Request"))
	mockPublisher.AssertCalled(t, "Publish", mock.Anything, sessionID, mock.AnythingOfType("pubsub.RequestEvent"))

	assert.Equal(t, 1, proxyServer1HitCount, "Proxy server 1 should have been hit once")
	assert.Equal(t, 1, proxyServer2HitCount, "Proxy server 2 should have been hit once")

	assert.NotNil(t, capturedRequest)
	assert.Len(t, capturedRequest.ForwardedRequests, 2, "Should have two forwarded requests recorded")

	// Assertions for ForwardedRequest 1
	fr1 := capturedRequest.ForwardedRequests[0]
	assert.Equal(t, proxyServer1.URL+"/proxy1", fr1.URL)
	assert.Equal(t, http.StatusOK, fr1.StatusCode)
	assert.Equal(t, `{"proxy_ok":1}`, string(fr1.ResponseBody))
	assert.Contains(t, fr1.ResponseHeaders, storage.HttpHeader{Name: "X-Proxy-Resp", Value: "proxy1-ok"})
	assert.Equal(t, requestBody, string(fr1.RequestBody))
	assert.Contains(t, fr1.RequestHeaders, storage.HttpHeader{Name: "Content-Type", Value: "application/json"})
	assert.Contains(t, fr1.RequestHeaders, storage.HttpHeader{Name: "X-Test-Header", Value: "test-value"})
	assert.Empty(t, fr1.Error)

	// Assertions for ForwardedRequest 2
	fr2 := capturedRequest.ForwardedRequests[1]
	assert.Equal(t, proxyServer2.URL+"/proxy2", fr2.URL)
	assert.Equal(t, http.StatusAccepted, fr2.StatusCode)
	assert.Equal(t, `{"proxy_ok":2}`, string(fr2.ResponseBody))
	assert.Contains(t, fr2.ResponseHeaders, storage.HttpHeader{Name: "X-Proxy-Resp", Value: "proxy2-ok"})
	assert.Equal(t, requestBody, string(fr2.RequestBody))
	assert.Empty(t, fr2.Error)

	mockDB.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestWebhookMiddleware_NoProxyURLsConfigured(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	mockDB := new(MockStorage)
	mockPublisher := new(MockPubSub)
	appCfg := &config.AppSettings{
		SessionTTL:         time.Hour,
		MaxRequestBodySize: 1024,
	}

	sessionID := "test-session-no-proxy"
	requestBody := `{"key":"value"}`
	appResponseBody := `{"app_says":"no_proxy_test"}`

	mockSession := &storage.Session{
		UUID:         sessionID,
		Code:         http.StatusCreated,
		ResponseBody: []byte(appResponseBody),
		Headers:      []storage.HttpHeader{{Name: "Content-Type", Value: "application/json"}},
		ProxyURLs:    []string{}, // Empty proxy URLs
		ProxyResponseMode: "app_response",
		CreatedAtUnixMilli: time.Now().UnixMilli(),
	}

	mockDB.On("GetSession", mock.Anything, sessionID).Return(mockSession, nil)
	mockDB.On("AddSessionTTL", mock.Anything, sessionID, mock.Anything).Return(nil)

	var capturedRequest storage.Request
	mockDB.On("NewRequest", mock.Anything, sessionID, mock.AnythingOfType("storage.Request")).Run(func(args mock.Arguments) {
		capturedRequest = args.Get(2).(storage.Request)
	}).Return("test-req-id-no-proxy", nil)

	mockDB.On("GetRequest", mock.Anything, sessionID, "test-req-id-no-proxy").Return(&storage.Request{
		UUID: "test-req-id-no-proxy",
		CreatedAtUnixMilli: time.Now().UnixMilli(),
	}, nil)
	mockPublisher.On("Publish", mock.Anything, sessionID, mock.AnythingOfType("pubsub.RequestEvent")).Return(nil)

	middleware := New(context.Background(), logger, mockDB, mockPublisher, appCfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("PUT", "/"+sessionID+"/test", strings.NewReader(requestBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, appResponseBody, rr.Body.String())

	mockDB.AssertCalled(t, "GetSession", mock.Anything, sessionID)
	mockDB.AssertCalled(t, "NewRequest", mock.Anything, sessionID, mock.AnythingOfType("storage.Request"))
	
	assert.NotNil(t, capturedRequest)
	assert.Empty(t, capturedRequest.ForwardedRequests, "ForwardedRequests should be empty")

	mockDB.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}


func TestWebhookMiddleware_ProxyRequestError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	mockDB := new(MockStorage)
	mockPublisher := new(MockPubSub)
	appCfg := &config.AppSettings{
		SessionTTL: time.Hour,
		MaxRequestBodySize: 1024,
	}

	// --- Mock Proxy Server (will simulate error by being closed immediately) ---
	proxyServerErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not be hit if server is down, or return 500 if hit
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	proxyServerErrUrl := proxyServerErr.URL // Save URL before closing
	proxyServerErr.Close() // Close server to simulate connection error
	
	// --- Second Proxy Server (successful) ---
	proxyServerOkHitCount := 0
	proxyServerOk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyServerOkHitCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer proxyServerOk.Close()


	sessionID := "test-session-proxy-error"
	requestBody := `{"data":"sendme"}`
	appResponseBody := `{"app_error_test":"done"}`

	mockSession := &storage.Session{
		UUID:         sessionID,
		Code:         http.StatusOK,
		ResponseBody: []byte(appResponseBody),
		ProxyURLs:    []string{proxyServerErrUrl + "/error_path", proxyServerOk.URL + "/ok_path"},
		ProxyResponseMode: "app_response",
		CreatedAtUnixMilli: time.Now().UnixMilli(),
	}

	mockDB.On("GetSession", mock.Anything, sessionID).Return(mockSession, nil)
	mockDB.On("AddSessionTTL", mock.Anything, sessionID, mock.Anything).Return(nil)

	var capturedRequest storage.Request
	mockDB.On("NewRequest", mock.Anything, sessionID, mock.AnythingOfType("storage.Request")).Run(func(args mock.Arguments) {
		capturedRequest = args.Get(2).(storage.Request)
	}).Return("test-req-id-proxy-err", nil)

	mockDB.On("GetRequest", mock.Anything, sessionID, "test-req-id-proxy-err").Return(&storage.Request{
		UUID: "test-req-id-proxy-err",
		CreatedAtUnixMilli: time.Now().UnixMilli(),
	}, nil)
	mockPublisher.On("Publish", mock.Anything, sessionID, mock.AnythingOfType("pubsub.RequestEvent")).Return(nil)

	middleware := New(context.Background(), logger, mockDB, mockPublisher, appCfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("POST", "/"+sessionID+"/error_test", strings.NewReader(requestBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code, "Response should be from app, not proxy")
	assert.Equal(t, appResponseBody, rr.Body.String())
	
	mockDB.AssertCalled(t, "NewRequest", mock.Anything, sessionID, mock.AnythingOfType("storage.Request"))

	assert.NotNil(t, capturedRequest)
	assert.Len(t, capturedRequest.ForwardedRequests, 2, "Should have two forwarded request entries")

	// Assertions for ForwardedRequest 1 (Error expected)
	fr1 := capturedRequest.ForwardedRequests[0]
	assert.Equal(t, proxyServerErrUrl+"/error_path", fr1.URL)
	assert.NotEmpty(t, fr1.Error, "Error field should be populated for the failed proxy request")
	assert.Equal(t, 0, fr1.StatusCode, "Status code should be 0 or unset for a connection error")
	assert.Equal(t, requestBody, string(fr1.RequestBody))


	// Assertions for ForwardedRequest 2 (Success expected)
	fr2 := capturedRequest.ForwardedRequests[1]
	assert.Equal(t, proxyServerOk.URL+"/ok_path", fr2.URL)
	assert.Empty(t, fr2.Error, "Error field should be empty for successful proxy request")
	assert.Equal(t, http.StatusOK, fr2.StatusCode)
	assert.Equal(t, `{"ok":true}`, string(fr2.ResponseBody))
	assert.Equal(t, requestBody, string(fr2.RequestBody))
	assert.Equal(t, 1, proxyServerOkHitCount, "Successful proxy server should have been hit")


	mockDB.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

// Helper to find a specific header in a slice of HttpHeader
func findHeader(headers []storage.HttpHeader, name string) (string, bool) {
	for _, h := range headers {
		if h.Name == name {
			return h.Value, true
		}
	}
	return "", false
}

func TestMain(m *testing.M) {
	// For tests that use openapi.IsValidUUID, we need to ensure it's set.
	// This is a simple stub. In a real scenario, this might be handled by build tags or a different setup.
	openapi.UUIDLength = 36
	openapi.IsValidUUID = func(u string) bool { return len(u) == openapi.UUIDLength } // Simplified stub
	
	m.Run()
}
