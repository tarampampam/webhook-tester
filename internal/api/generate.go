package api

import (
	_ "github.com/deepmap/oapi-codegen/pkg/runtime"
	_ "github.com/deepmap/oapi-codegen/pkg/types"
)

// Generate openapi stubs (`oapi-codegen` is required for this):
//go:generate oapi-codegen -generate server,types -o ./openapi.gen.go -package api ./../../api/openapi.yml
