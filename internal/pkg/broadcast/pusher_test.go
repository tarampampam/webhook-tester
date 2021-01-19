package broadcast

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req), nil }

func TestBroadcaster_Publish(t *testing.T) {
	var catch = false

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			catch = true

			body, err := ioutil.ReadAll(req.Body)
			assert.Nil(t, err)

			assert.Contains(t, "https://api-yeah.pusher.com/", req.RequestURI)
			assert.JSONEq(t, `{"name":"request-registered","channels":["channel"],"data":"bar"}`, string(body))

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
			}
		}),
	}

	broadcaster := NewPusher("foo", "bar", "baz", "yeah", WithPusherHTTPClient(client))

	assert.Nil(t, broadcaster.Publish("channel", NewRequestRegisteredEvent("bar")))
	assert.True(t, catch)
}
