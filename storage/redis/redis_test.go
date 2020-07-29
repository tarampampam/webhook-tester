package redis

import (
	"testing"
	"time"
	"webhook-tester/storage"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

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
			s, miniRedis := setup(t)
			defer func() {
				assert.Nil(t, s.Close())
				miniRedis.Close()
			}()

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

func setup(t *testing.T) (*Storage, *miniredis.Miniredis) {
	miniRedis, err := miniredis.Run()

	assert.Nil(t, err)

	miniRedis.Select(0)

	s := NewStorage("", "", 0, 1, time.Second*10, 1)
	s.redis = redis.NewClient(&redis.Options{
		Addr:     miniRedis.Addr(),
		Username: "",
		Password: "",
		DB:       0,
		PoolSize: 1,
	})

	return s, miniRedis
}
