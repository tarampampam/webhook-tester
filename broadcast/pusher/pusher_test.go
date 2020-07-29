package pusher

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req), nil }

func TestBroadcaster_Publish(t *testing.T) {
	var catch bool = false

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			catch = true

			body, err := ioutil.ReadAll(req.Body)
			assert.Nil(t, err)

			assert.Contains(t, "https://api-yeah.pusher.com/", req.RequestURI)
			assert.JSONEq(t, `{"name":"eventName","channels":["channel"],"data":"data"}`, string(body))

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
			}
		}),
	}

	broadcaster := NewBroadcaster("foo", "bar", "baz", "yeah")
	broadcaster.pusher.HTTPClient = client

	assert.Nil(t, broadcaster.Publish("channel", "eventName", "data"))
	assert.True(t, catch)
}

func TestNewBroadcaster(t *testing.T) {
	t.Parallel()

	broadcaster := NewBroadcaster("foo", "bar", "baz", "yeah")

	assert.Equal(t, "foo", broadcaster.pusher.AppID)
	assert.Equal(t, "bar", broadcaster.pusher.Key)
	assert.Equal(t, "baz", broadcaster.pusher.Secret)
	assert.Equal(t, "yeah", broadcaster.pusher.Cluster)
	assert.True(t, broadcaster.pusher.Secure)
}
