package api

import (
	_ "github.com/oapi-codegen/runtime"
	_ "github.com/oapi-codegen/runtime/types"
)

// Generate openapi stubs (`oapi-codegen` is required for this):
//go:generate oapi-codegen -generate server,types -o ./openapi.gen.go -package api ./../../api/openapi.yml
