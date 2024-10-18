package version

import "gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"

type Handler struct{ ver string }

func New(ver string) *Handler { return &Handler{ver: ver} }

func (h *Handler) Handle() openapi.VersionResponse { return openapi.VersionResponse{Version: h.ver} }
