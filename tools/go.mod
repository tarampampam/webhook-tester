// This module tracks dev tools that are not part of the main module's dependency graph.
// Run all commands from the repository root. The -C flag (Go 1.21+) changes the working
// directory before execution; -modfile points go get at this file without changing cwd.
//
// Add a new tool (adds to both `tool` and `require`):
//   go get -modfile=tools/go.mod -tool <pkg>@latest
//
// Remove a tool:
//   go get -modfile=tools/go.mod -tool <pkg>@none
//
// Update a specific tool:
//   go get -modfile=tools/go.mod -tool <pkg>@latest
//
// Run a tool:
//   go tool -modfile=tools/go.mod <tool> [args...]

module tools

go 1.27

tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

require (
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/getkin/kin-openapi v0.135.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.4 // indirect
	github.com/go-openapi/swag/jsonname v0.25.4 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.7.0 // indirect
	github.com/oasdiff/yaml v0.0.9 // indirect
	github.com/oasdiff/yaml3 v0.0.9 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/speakeasy-api/jsonpath v0.6.3 // indirect
	github.com/speakeasy-api/openapi v1.19.2 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	github.com/woodsbury/decimal128 v1.4.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
