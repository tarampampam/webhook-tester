package http

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type API struct {
	ctx  context.Context
	cfg  config.Config
	rdb  *redis.Client
	stor storage.Storage
	pub  pubsub.Publisher
	sub  pubsub.Subscriber
	reg  *prometheus.Registry

	apiVersion
	apiHealth
}

var _ api.ServerInterface = (*API)(nil) // verify that API implements interface

func (*API) ApiSessionCreate(c echo.Context) error {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionDelete(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionDeleteAllRequests(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionGetAllRequests(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionDeleteRequest(c echo.Context, session api.SessionUUID, request api.RequestUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionGetRequest(c echo.Context, session api.SessionUUID, request api.RequestUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSettings(c echo.Context) error {
	// TODO implement me
	panic("implement me")
}

func (*API) AppMetrics(c echo.Context) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebsocketSession(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookDelete(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookGet(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookHead(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookOptions(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookPatch(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookPost(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookPut(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookTrace(c echo.Context, session api.SessionUUID) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyDelete(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyGet(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyHead(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyOptions(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyPatch(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyPost(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyPut(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyTrace(c echo.Context, session api.SessionUUID, any string) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeDelete(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeGet(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeHead(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeOptions(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodePatch(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodePost(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodePut(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeTrace(c echo.Context, session api.SessionUUID, status api.RequiredStatusCode) error {
	// TODO implement me
	panic("implement me")
}
