package api_test

import (
	"testing"

	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"

	"github.com/stretchr/testify/assert"
)

func TestNewServerError(t *testing.T) {
	model := api.NewServerError(1, "foo")

	assert.False(t, model.Success)
	assert.Equal(t, 1, model.Code)
	assert.Equal(t, "foo", model.Message)
}

func TestServerError_ToJSON(t *testing.T) {
	model := api.NewServerError(123, "foo")

	asJSON, err := model.ToJSON()
	assert.NoError(t, err)

	assert.JSONEq(t, `{"code":123,"success":false,"message":"foo"}`, string(asJSON))
}
