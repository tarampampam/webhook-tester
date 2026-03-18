package requests_list_test

import (
	"context"
	"encoding/base64"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/handlers/requests_list"
	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

func uint32Ptr(v uint32) *uint32 { return &v }
func boolPtr(v bool) *bool       { return &v }

// setupTestData creates a session with the given number of requests using controlled timestamps.
// Each request gets a timestamp of baseTime + i seconds, ensuring deterministic ordering.
func setupTestData(t *testing.T, ctx context.Context, db *storage.InMemory, count int) string {
	t.Helper()

	sID, err := db.NewSession(ctx, storage.Session{Code: 200})
	require.NoError(t, err)

	for i := range count {
		_, err = db.NewRequest(ctx, sID, storage.Request{
			ClientAddr: "127.0.0.1",
			Method:     "POST",
			Body:       []byte("body-" + string(rune('a'+i))),
			Headers:    []storage.HttpHeader{{Name: "Content-Type", Value: "text/plain"}},
			URL:        "http://example.com/hook",
		})
		require.NoError(t, err)
	}

	return sID
}

// newTestDB creates an InMemory storage with a controlled clock that increments by 1 second per call.
func newTestDB(t *testing.T) *storage.InMemory {
	t.Helper()

	var counter atomic.Int64

	db := storage.NewInMemory(time.Minute, 128, storage.WithInMemoryTimeNow(func() time.Time {
		n := counter.Add(1)
		return time.Unix(n, 0)
	}))

	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}

func TestHandle_NoParams(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 5)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{})
	require.NoError(t, err)
	require.Len(t, *resp, 5)

	// verify sorted newest first
	for i := 1; i < len(*resp); i++ {
		assert.Greater(t, (*resp)[i-1].CapturedAtUnixMilli, (*resp)[i].CapturedAtUnixMilli,
			"requests should be sorted newest first")
	}

	// verify body is included by default
	for _, r := range *resp {
		assert.NotEmpty(t, r.RequestPayloadBase64, "body should be included by default")
	}
}

func TestHandle_Limit(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 5)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		Limit: uint32Ptr(2),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 2)

	// should be the 2 newest
	allResp, _ := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{})
	assert.Equal(t, (*allResp)[0].CapturedAtUnixMilli, (*resp)[0].CapturedAtUnixMilli)
	assert.Equal(t, (*allResp)[1].CapturedAtUnixMilli, (*resp)[1].CapturedAtUnixMilli)
}

func TestHandle_LimitExceedsTotal(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 3)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		Limit: uint32Ptr(100),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 3, "should return all when limit exceeds total")
}

func TestHandle_LimitZero(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 3)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		Limit: uint32Ptr(0),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 3, "limit=0 should return all (same as omitted)")
}

func TestHandle_Offset(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 5)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	allResp, _ := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{})

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		Offset: uint32Ptr(2),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 3, "should skip 2 newest, return remaining 3")

	// the 3rd newest should now be first
	assert.Equal(t, (*allResp)[2].CapturedAtUnixMilli, (*resp)[0].CapturedAtUnixMilli)
}

func TestHandle_OffsetExceedsTotal(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 3)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		Offset: uint32Ptr(100),
	})
	require.NoError(t, err)
	require.Empty(t, *resp, "offset beyond total should return empty")
}

func TestHandle_LimitAndOffset(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 5)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	allResp, _ := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{})

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		Limit:  uint32Ptr(2),
		Offset: uint32Ptr(1),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 2, "should skip 1 newest, return next 2")

	assert.Equal(t, (*allResp)[1].CapturedAtUnixMilli, (*resp)[0].CapturedAtUnixMilli)
	assert.Equal(t, (*allResp)[2].CapturedAtUnixMilli, (*resp)[1].CapturedAtUnixMilli)
}

func TestHandle_IncludeBodyFalse(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 3)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		IncludeBody: boolPtr(false),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 3)

	for _, r := range *resp {
		assert.Empty(t, r.RequestPayloadBase64, "body should be empty when include_body=false")
		assert.NotEmpty(t, r.Headers, "headers should still be present")
		assert.NotEmpty(t, r.Method, "method should still be present")
	}
}

func TestHandle_IncludeBodyTrue(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 2)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		IncludeBody: boolPtr(true),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 2)

	for _, r := range *resp {
		decoded, err := base64.StdEncoding.DecodeString(r.RequestPayloadBase64)
		require.NoError(t, err)
		assert.Contains(t, string(decoded), "body-", "body should be present when include_body=true")
	}
}

func TestHandle_EmptySession(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	sID, err := db.NewSession(ctx, storage.Session{Code: 200})
	require.NoError(t, err)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{})
	require.NoError(t, err)
	require.Empty(t, *resp, "empty session should return empty list")
}

func TestHandle_SessionNotFound(t *testing.T) {
	db := newTestDB(t)

	handler := requests_list.New(db)
	fakeUUID, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)

	_, err = handler.Handle(context.Background(), fakeUUID, openapi.ApiSessionListRequestsParams{})
	require.Error(t, err)
}

func TestHandle_AllParamsCombined(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	sID := setupTestData(t, ctx, db, 10)

	handler := requests_list.New(db)
	sUUID, err := uuid.Parse(sID)
	require.NoError(t, err)

	allResp, _ := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{})
	require.Len(t, *allResp, 10)

	resp, err := handler.Handle(ctx, sUUID, openapi.ApiSessionListRequestsParams{
		Limit:       uint32Ptr(3),
		Offset:      uint32Ptr(2),
		IncludeBody: boolPtr(false),
	})
	require.NoError(t, err)
	require.Len(t, *resp, 3)

	// skip 2 newest, take next 3
	assert.Equal(t, (*allResp)[2].CapturedAtUnixMilli, (*resp)[0].CapturedAtUnixMilli)
	assert.Equal(t, (*allResp)[3].CapturedAtUnixMilli, (*resp)[1].CapturedAtUnixMilli)
	assert.Equal(t, (*allResp)[4].CapturedAtUnixMilli, (*resp)[2].CapturedAtUnixMilli)

	for _, r := range *resp {
		assert.Empty(t, r.RequestPayloadBase64, "body should be empty")
	}
}
