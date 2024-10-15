package storage_test

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/internal/storage"
)

func TestTime_JSON_Marshal_Unmarshal(t *testing.T) {
	t.Parallel()

	type someStruct struct {
		Time storage.Time `json:"time"`
	}

	t.Run("common case", func(t *testing.T) {
		var (
			someTime      = time.Date(2021, 1, 1, 2, 3, 4, 5, time.Local)
			originalValue = someStruct{Time: storage.Time{Time: someTime}}
		)

		marshaled, err := json.Marshal(originalValue)
		require.NoError(t, err)
		require.Equal(t, `{"time":1609452184000000005}`, string(marshaled))

		var unmarshalled = someStruct{}

		require.NoError(t, json.Unmarshal(marshaled, &unmarshalled))
		require.Equal(t, originalValue, unmarshalled)
	})

	t.Run("zero value", func(t *testing.T) {
		var (
			zeroValue     storage.Time
			originalValue = someStruct{Time: zeroValue}
		)

		marshaled, err := json.Marshal(originalValue)
		require.NoError(t, err)
		require.Equal(t, `{"time":0}`, string(marshaled))

		var unmarshalled = someStruct{}

		require.NoError(t, json.Unmarshal(marshaled, &unmarshalled))
		require.Equal(t, originalValue, unmarshalled)
	})
}

func TestURL_JSON_Marshal_Unmarshal(t *testing.T) {
	t.Parallel()

	type someStruct struct {
		URL storage.URL `json:"url"`
	}

	t.Run("common case", func(t *testing.T) {
		var (
			someUrl, _    = url.Parse("https://example.com/path?query=value#fragment")
			originalValue = someStruct{URL: storage.URL{URL: *someUrl}}
		)

		marshaled, err := json.Marshal(originalValue)
		require.NoError(t, err)
		require.Equal(t, `{"url":"https://example.com/path?query=value#fragment"}`, string(marshaled))

		var unmarshalled = someStruct{}

		require.NoError(t, json.Unmarshal(marshaled, &unmarshalled))
		require.Equal(t, originalValue, unmarshalled)
	})

	t.Run("zero value", func(t *testing.T) {
		var (
			zeroValue     storage.URL
			originalValue = someStruct{URL: zeroValue}
		)

		marshaled, err := json.Marshal(originalValue)
		require.NoError(t, err)
		require.Equal(t, `{"url":""}`, string(marshaled))

		var unmarshalled = someStruct{}

		require.NoError(t, json.Unmarshal(marshaled, &unmarshalled))
		require.Equal(t, originalValue, unmarshalled)
	})
}
