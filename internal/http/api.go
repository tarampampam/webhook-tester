package http

import (
	"context"
	"net/http"

	"github.com/go-redis/redis/v8"
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
}

var _ api.ServerInterface = (*API)(nil) // verify that API implements interface

func (*API) ApiSessionCreate(w http.ResponseWriter, r *http.Request) {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionDelete(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionDeleteAllRequests(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionGetAllRequests(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionDeleteRequest(w http.ResponseWriter, r *http.Request, session api.SessionUUID, request api.RequestUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSessionGetRequest(w http.ResponseWriter, r *http.Request, session api.SessionUUID, request api.RequestUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiSettings(w http.ResponseWriter, r *http.Request) {
	// TODO implement me
	panic("implement me")
}

func (*API) ApiAppVersion(w http.ResponseWriter, r *http.Request) {
	// TODO implement me
	panic("implement me")
}

func (*API) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	// TODO implement me
	panic("implement me")
}

func (*API) AppMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO implement me
	panic("implement me")
}

func (*API) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebsocketSession(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookDelete(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookGet(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookHead(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookOptions(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookPatch(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookPost(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookPut(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookTrace(w http.ResponseWriter, r *http.Request, session api.SessionUUID) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyDelete(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyGet(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyHead(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyOptions(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyPatch(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyPost(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyPut(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithAnyTrace(w http.ResponseWriter, r *http.Request, session api.SessionUUID, any string) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeDelete(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeGet(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeHead(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeOptions(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodePatch(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodePost(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodePut(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}

func (*API) WebhookWithCodeTrace(w http.ResponseWriter, r *http.Request, session api.SessionUUID, status api.RequiredStatusCode) {
	// TODO implement me
	panic("implement me")
}
