package openapi

import (
	_ "github.com/oapi-codegen/runtime" // required for oapi-codegen
	_ "github.com/oapi-codegen/runtime/types"
)

// Generate openapi stubs (`oapi-codegen` is required for this):
//go:generate go tool -modfile=../../../tools.go.mod oapi-codegen -config ./configs/models.yml -o ./models.gen.go -package openapi ./../../../api/openapi.yml
//go:generate go tool -modfile=../../../tools.go.mod oapi-codegen -config ./configs/server.yml -o ./server.gen.go -package openapi ./../../../api/openapi.yml
//go:generate go tool -modfile=../../../tools.go.mod oapi-codegen -config ./configs/spec.yml   -o ./spec.gen.go   -package openapi ./../../../api/openapi.yml
