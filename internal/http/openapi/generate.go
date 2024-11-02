package openapi

import _ "github.com/oapi-codegen/runtime/types" // required for oapi-codegen

// Generate openapi stubs (`oapi-codegen` is required for this):
//go:generate oapi-codegen -config ./configs/models.yml -o ./models.gen.go -package openapi ./../../../api/openapi.yml
//go:generate oapi-codegen -config ./configs/server.yml -o ./server.gen.go -package openapi ./../../../api/openapi.yml
//go:generate oapi-codegen -config ./configs/spec.yml   -o ./spec.gen.go   -package openapi ./../../../api/openapi.yml
