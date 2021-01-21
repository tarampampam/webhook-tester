package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInMemoryStorage_SessionCreateReadDelete(t *testing.T) {
	s := NewInMemoryStorage(time.Minute, 1, time.Second)
	defer s.Close()

	sessionUUID, creationErr := s.CreateSession("foo bar", 201, "text/javascript", time.Second*123)
	assert.NoError(t, creationErr)

	noSession, noSessionErr := s.GetSession("foo")
	assert.Nil(t, noSession)
	assert.NoError(t, noSessionErr)

	gotSession, gotSessionErr := s.GetSession(sessionUUID)

	assert.NoError(t, gotSessionErr)
	assert.Equal(t, sessionUUID, gotSession.UUID())
	assert.Equal(t, time.Now().Unix(), gotSession.CreatedAt().Unix())
	assert.Equal(t, (time.Second*123).Nanoseconds(), gotSession.Delay().Nanoseconds())
	assert.Equal(t, "text/javascript", gotSession.ContentType())
	assert.Equal(t, "foo bar", gotSession.Content())
	assert.Equal(t, uint16(201), gotSession.Code())
	assert.Equal(t, sessionUUID, gotSession.UUID())

	delNonExists, errDelNonExists := s.DeleteSession("foo")
	assert.False(t, delNonExists)
	assert.NoError(t, errDelNonExists)

	delExists, errDelExists := s.DeleteSession(sessionUUID)
	assert.True(t, delExists)
	assert.NoError(t, errDelExists)

	gotSessionAgain, gotSessionAgainErr := s.GetSession(sessionUUID)
	assert.Nil(t, gotSessionAgain)
	assert.NoError(t, gotSessionAgainErr)
}

func TestInMemoryStorage_RequestCreateReadDelete(t *testing.T) {
	s := NewInMemoryStorage(time.Minute, 10, time.Nanosecond * 100)
	defer s.Close()

	sessionUUID, sessionCreationErr := s.CreateSession("foo bar", 201, "text/javascript", 0)
	assert.Nil(t, sessionCreationErr)

	requestUUID, creationErr := s.CreateRequest(
		sessionUUID,
		"2.3.4.5",
		"GET",
		`{"foo":123}`,
		"https://example.com/test",
		map[string]string{"foo": "bar"},
	)
	assert.Nil(t, creationErr)
	assert.NotEmpty(t, requestUUID)

	noRequest, noRequestErr := s.GetRequest(sessionUUID, "foo")
	assert.Nil(t, noRequest)
	assert.Nil(t, noRequestErr)

	request, getRequestErr := s.GetRequest(sessionUUID, requestUUID)
	assert.Nil(t, getRequestErr)
	assert.Equal(t, "2.3.4.5", request.ClientAddr())
	assert.Equal(t, `{"foo":123}`, request.Content())
	assert.Equal(t, map[string]string{"foo": "bar"}, request.Headers())
	assert.Equal(t, "https://example.com/test", request.URI())
	assert.Equal(t, "GET", request.Method())
	assert.Equal(t, time.Now().Unix(), request.CreatedAt().Unix())
	assert.Equal(t, requestUUID, request.UUID())

	noDelResult, noDelErr := s.DeleteRequest(sessionUUID, "foo")
	assert.False(t, noDelResult)
	assert.NoError(t, noDelErr)

	delResult, delErr := s.DeleteRequest(sessionUUID, requestUUID)
	assert.True(t, delResult)
	assert.NoError(t, delErr)

	nowNoRequest, nowNoRequestErr := s.GetRequest(sessionUUID, requestUUID)
	assert.Nil(t, nowNoRequest)
	assert.NoError(t, nowNoRequestErr)
}

func TestInMemoryStorage_RequestCreationLimit(t *testing.T) {
	s := NewInMemoryStorage(time.Minute, 2, time.Nanosecond * 100)
	defer s.Close()

	sessionUUID, _ := s.CreateSession("foo bar", 201, "text/javascript", 0)

	_, _ = s.CreateRequest(sessionUUID, "1.1.1.1", "GET", `{"foo":123}`, "https://example.com/test", nil)

	requests, _ := s.GetAllRequests(sessionUUID)
	assert.Len(t, requests, 1)

	_, _ = s.CreateRequest(sessionUUID, "2.2.2.2", "GET", `{"foo":123}`, "https://example.com/test", nil)

	requests, _ = s.GetAllRequests(sessionUUID)
	assert.Len(t, requests, 2)
	assert.Equal(t, "1.1.1.1", requests[0].ClientAddr())
	assert.Equal(t, "2.2.2.2", requests[1].ClientAddr())

	_, _ = s.CreateRequest(sessionUUID, "3.3.3.3", "GET", `{"foo":123}`, "https://example.com/test", nil)

	requests, _ = s.GetAllRequests(sessionUUID)
	assert.Len(t, requests, 2)
	assert.Equal(t, "2.2.2.2", requests[0].ClientAddr())
	assert.Equal(t, "3.3.3.3", requests[1].ClientAddr())
}

func TestInMemoryStorage_GetAllRequests(t *testing.T) {
	s := NewInMemoryStorage(time.Minute, 10, time.Nanosecond * 100)
	defer s.Close()

	sessionUUID, sessionCreationErr := s.CreateSession("foo bar", 201, "text/javascript", 0)
	assert.NoError(t, sessionCreationErr)

	noRequests, noRequestsErr := s.GetAllRequests(sessionUUID)
	assert.Nil(t, noRequests)
	assert.NoError(t, noRequestsErr)

	noRequestsWrongSession, noRequestsWrongSessionErr := s.GetAllRequests("foo")
	assert.Nil(t, noRequestsWrongSession)
	assert.NoError(t, noRequestsWrongSessionErr)

	requestUUID, creationErr := s.CreateRequest(sessionUUID, "1.2.3.4", "GET", `{"foo":123}`, "https://test", nil)
	assert.NoError(t, creationErr)
	assert.NotEmpty(t, requestUUID)

	requests, requestsErr := s.GetAllRequests(sessionUUID)
	assert.Len(t, requests, 1)
	assert.NoError(t, requestsErr)
	assert.Equal(t, "1.2.3.4", requests[0].ClientAddr())
}

func TestInMemoryStorage_DeleteRequests(t *testing.T) {
	s := NewInMemoryStorage(time.Minute, 10, time.Nanosecond * 100)
	defer s.Close()

	sessionUUID, sessionCreationErr := s.CreateSession("foo bar", 201, "text/javascript", 0)
	assert.NoError(t, sessionCreationErr)

	res, delErr := s.DeleteRequests(sessionUUID)
	assert.False(t, res)
	assert.NoError(t, delErr)

	_, _ = s.CreateRequest(sessionUUID, "1.1.1.1", "GET", `{"foo":123}`, "https://test", nil)
	_, _ = s.CreateRequest(sessionUUID, "1.1.1.1", "GET", `{"foo":123}`, "https://test", nil)

	requests, _ := s.GetAllRequests(sessionUUID)
	assert.Len(t, requests, 2)

	res2, delErr2 := s.DeleteRequests(sessionUUID)
	assert.True(t, res2)
	assert.NoError(t, delErr2)

	requests2, _ := s.GetAllRequests(sessionUUID)
	assert.Nil(t, requests2)
}

func TestInMemoryStorage_GetSessionExpired(t *testing.T) {
	s := NewInMemoryStorage(time.Millisecond * 10, 10, time.Minute)
	defer s.Close()

	sessionUUID, err := s.CreateSession("foo bar", 201, "text/javascript", 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, sessionUUID)

	session, err := s.GetSession(sessionUUID)
	assert.NoError(t, err)
	assert.NotNil(t, session)

	<-time.After(time.Millisecond * 11)

	session, err = s.GetSession(sessionUUID)
	assert.NoError(t, err)
	assert.Nil(t, session) // important
}
