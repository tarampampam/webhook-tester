package null

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	st "github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func TestStorage_All(t *testing.T) {
	requests := make([]st.RequestData, 1)
	requests = append(requests, st.RequestData{})

	s := Storage{
		SessionData: &st.SessionData{},
		RequestData: &st.RequestData{},
		Requests:    &requests,
		Error:       errors.New(""),
		Boolean:     true,
	}

	assert.Equal(t, s.Error, s.Close())

	delSess, delSessErr := s.DeleteSession("")
	assert.Equal(t, s.Boolean, delSess)
	assert.Same(t, s.Error, delSessErr)

	delRequests, delRequestsErr := s.DeleteRequests("")
	assert.Equal(t, s.Boolean, delRequests)
	assert.Same(t, s.Error, delRequestsErr)

	delRequest, delRequestErr := s.DeleteRequest("", "")
	assert.Equal(t, s.Boolean, delRequest)
	assert.Same(t, s.Error, delRequestErr)

	gotSess, gotSessErr := s.GetSession("")
	assert.Same(t, s.SessionData, gotSess)
	assert.Same(t, s.Error, gotSessErr)

	gotRequest, gotRequestErr := s.GetRequest("", "")
	assert.Same(t, s.RequestData, gotRequest)
	assert.Same(t, s.Error, gotRequestErr)

	gotAllRequests, gotAllRequestsErr := s.GetAllRequests("")
	assert.Same(t, s.Requests, gotAllRequests)
	assert.Same(t, s.Error, gotAllRequestsErr)

	newSess, newSessErr := s.CreateSession(&st.WebHookResponse{})
	assert.Same(t, s.SessionData, newSess)
	assert.Same(t, s.Error, newSessErr)

	newReq, newReqErr := s.CreateRequest("", &st.Request{})
	assert.Same(t, s.RequestData, newReq)
	assert.Same(t, s.Error, newReqErr)
}
