package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func TestStorage_SessionCreateReadDelete(t *testing.T) {
	t.Parallel()

	s, miniRedis := newFakeRedisStorage(t, 1)

	defer miniRedis.Close()

	wh := &storage.WebHookResponse{
		Content:     "foo bar",
		Code:        201,
		ContentType: "text/javascript",
		DelaySec:    5,
	}
	sessionData, creationErr := s.CreateSession(wh)
	assert.Nil(t, creationErr)

	noSession, noSessionErr := s.GetSession("foo")
	assert.Nil(t, noSession)
	assert.Nil(t, noSessionErr)

	gotSession, gotSessionErr := s.GetSession(sessionData.UUID)
	assert.Nil(t, gotSessionErr)
	assert.Equal(t, sessionData.UUID, gotSession.UUID)
	assert.Equal(t, time.Now().Unix(), gotSession.CreatedAtUnix)
	assert.Equal(t, wh.DelaySec, gotSession.WebHookResponse.DelaySec)
	assert.Equal(t, wh.ContentType, gotSession.WebHookResponse.ContentType)
	assert.Equal(t, wh.Content, gotSession.WebHookResponse.Content)
	assert.Equal(t, wh.Code, gotSession.WebHookResponse.Code)

	delNonExists, errDelNonExists := s.DeleteSession("foo")
	assert.False(t, delNonExists)
	assert.Nil(t, errDelNonExists)

	delExists, errDelExists := s.DeleteSession(gotSession.UUID)
	assert.True(t, delExists)
	assert.Nil(t, errDelExists)

	gotSessionAgain, gotSessionAgainErr := s.GetSession(sessionData.UUID)
	assert.Nil(t, gotSessionAgain)
	assert.Nil(t, gotSessionAgainErr)
}

func TestStorage_RequestCreateReadDelete(t *testing.T) {
	t.Parallel()

	s, miniRedis := newFakeRedisStorage(t, 10)

	defer miniRedis.Close()

	session, sessionCreationErr := s.CreateSession(&storage.WebHookResponse{})
	assert.Nil(t, sessionCreationErr)

	r := &storage.Request{
		ClientAddr: "2.3.4.5",
		Method:     "GET",
		Content:    `{"foo":123}`,
		Headers:    map[string]string{"foo": "bar"},
		URI:        "https://example.com/test",
	}

	requestData, creationErr := s.CreateRequest(session.UUID, r)
	assert.Nil(t, creationErr)
	assert.NotNil(t, requestData)

	noRequest, noRequestErr := s.GetRequest(session.UUID, "foo")
	assert.Nil(t, noRequest)
	assert.Nil(t, noRequestErr)

	request, getRequestErr := s.GetRequest(session.UUID, requestData.UUID)
	assert.Nil(t, getRequestErr)
	assert.Equal(t, r.ClientAddr, request.Request.ClientAddr)
	assert.Equal(t, r.Content, request.Request.Content)
	assert.Equal(t, r.Headers, request.Request.Headers)
	assert.Equal(t, r.URI, request.Request.URI)
	assert.Equal(t, r.Method, request.Request.Method)

	noDelResult, noDelErr := s.DeleteRequest(session.UUID, "foo")
	assert.False(t, noDelResult)
	assert.Nil(t, noDelErr)

	delResult, delErr := s.DeleteRequest(session.UUID, requestData.UUID)
	assert.True(t, delResult)
	assert.Nil(t, delErr)

	nowNoRequest, nowNoRequestErr := s.GetRequest(session.UUID, requestData.UUID)
	assert.Nil(t, nowNoRequest)
	assert.Nil(t, nowNoRequestErr)
}

func TestStorage_RequestCreationLimit(t *testing.T) {
	t.Parallel()

	s, miniRedis := newFakeRedisStorage(t, 2)

	defer miniRedis.Close()

	session, _ := s.CreateSession(&storage.WebHookResponse{})

	var requests *[]storage.RequestData

	_, _ = s.CreateRequest(session.UUID, &storage.Request{ClientAddr: "1.1.1.1"})

	requests, _ = s.GetAllRequests(session.UUID)
	assert.Len(t, *requests, 1)

	_, _ = s.CreateRequest(session.UUID, &storage.Request{ClientAddr: "2.2.2.2"})

	requests, _ = s.GetAllRequests(session.UUID)
	assert.Len(t, *requests, 2)
	assert.Equal(t, "1.1.1.1", (*requests)[0].Request.ClientAddr)
	assert.Equal(t, "2.2.2.2", (*requests)[1].Request.ClientAddr)

	_, _ = s.CreateRequest(session.UUID, &storage.Request{ClientAddr: "3.3.3.3"})

	requests, _ = s.GetAllRequests(session.UUID)
	assert.Len(t, *requests, 2)
	assert.Equal(t, "2.2.2.2", (*requests)[0].Request.ClientAddr)
	assert.Equal(t, "3.3.3.3", (*requests)[1].Request.ClientAddr)
}

func TestStorage_GetAllRequests(t *testing.T) {
	t.Parallel()

	s, miniRedis := newFakeRedisStorage(t, 10)

	defer miniRedis.Close()

	session, sessionCreationErr := s.CreateSession(&storage.WebHookResponse{})
	assert.Nil(t, sessionCreationErr)

	noRequests, noRequestsErr := s.GetAllRequests(session.UUID)
	assert.Nil(t, noRequests)
	assert.Nil(t, noRequestsErr)

	noRequestsWrongSession, noRequestsWrongSessionErr := s.GetAllRequests("foo")
	assert.Nil(t, noRequestsWrongSession)
	assert.Nil(t, noRequestsWrongSessionErr)

	requestData, creationErr := s.CreateRequest(session.UUID, &storage.Request{ClientAddr: "1.2.3.4"})
	assert.Nil(t, creationErr)
	assert.NotNil(t, requestData)

	requests, requestsErr := s.GetAllRequests(session.UUID)
	assert.Len(t, *requests, 1)
	assert.Nil(t, requestsErr)
	assert.Equal(t, "1.2.3.4", (*requests)[0].Request.ClientAddr)
}

func TestStorage_DeleteRequests(t *testing.T) {
	t.Parallel()

	s, miniRedis := newFakeRedisStorage(t, 10)

	defer miniRedis.Close()

	session, sessionCreationErr := s.CreateSession(&storage.WebHookResponse{})
	assert.Nil(t, sessionCreationErr)

	res, delErr := s.DeleteRequests(session.UUID)
	assert.False(t, res)
	assert.Nil(t, delErr)

	_, _ = s.CreateRequest(session.UUID, &storage.Request{ClientAddr: "1.1.1.1"})
	_, _ = s.CreateRequest(session.UUID, &storage.Request{ClientAddr: "1.1.1.1"})

	requests, _ := s.GetAllRequests(session.UUID)
	assert.Len(t, *requests, 2)

	res2, delErr2 := s.DeleteRequests(session.UUID)
	assert.True(t, res2)
	assert.Nil(t, delErr2)

	requests2, _ := s.GetAllRequests(session.UUID)
	assert.Nil(t, requests2)
}

func TestStorage_GetSession(t *testing.T) {
	var (
		correctSessionJSON string = `{
			"resp_content":"foo bar",
			"resp_code":200,
			"resp_content_type":"text/plain",
			"resp_delay_sec":12,
			"created_at_unix":1596032211
		}`
		wrongJSON string = `{"foo"`
	)

	var cases = []struct {
		name            string
		giveSessionUUID string
		giveSessionKey  string
		giveSessionJSON *string
		checkFn         func(*testing.T, *storage.SessionData)
		wantError       bool
	}{
		{
			name:            "regular usage",
			giveSessionUUID: "094a0edf-12ad-4e08-8385-457f42513a38",
			giveSessionKey:  "webhook-tester:session:094a0edf-12ad-4e08-8385-457f42513a38",
			giveSessionJSON: &correctSessionJSON,
			checkFn: func(t *testing.T, s *storage.SessionData) {
				assert.Equal(t, "094a0edf-12ad-4e08-8385-457f42513a38", s.UUID)
				assert.Equal(t, "foo bar", s.WebHookResponse.Content)
				assert.Equal(t, uint16(200), s.WebHookResponse.Code)
				assert.Equal(t, "text/plain", s.WebHookResponse.ContentType)
				assert.Equal(t, uint8(12), s.WebHookResponse.DelaySec)
				assert.Equal(t, int64(1596032211), s.CreatedAtUnix)
			},
			wantError: false,
		},
		{
			name:            "non existing session",
			giveSessionUUID: "094a0edf-12ad-4e08-8385-457f42513a38",
			giveSessionKey:  "",
			giveSessionJSON: nil,
			checkFn: func(t *testing.T, s *storage.SessionData) {
				assert.Nil(t, s)
			},
			wantError: false,
		},
		{
			name:            "wrong json in storage",
			giveSessionUUID: "foo",
			giveSessionKey:  "webhook-tester:session:foo",
			giveSessionJSON: &wrongJSON,
			checkFn: func(t *testing.T, s *storage.SessionData) {
				assert.Nil(t, s)
			},
			wantError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, miniRedis := newFakeRedisStorage(t, 128)
			defer miniRedis.Close()

			if tt.giveSessionJSON != nil {
				assert.Nil(t, miniRedis.Set(tt.giveSessionKey, *tt.giveSessionJSON))
			}

			res, err := s.GetSession(tt.giveSessionUUID)

			if tt.checkFn != nil {
				tt.checkFn(t, res)
			}

			if tt.wantError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func newFakeRedisStorage(t *testing.T, maxRequests uint16) (*Storage, *miniredis.Miniredis) {
	miniRedis, err := miniredis.Run()

	assert.Nil(t, err)

	miniRedis.Select(0)

	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})

	s := NewStorage(context.TODO(), client, time.Second*10, maxRequests)
	s.rdb = redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	return s, miniRedis
}
