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
	github.com/getkin/kin-openapi v0.142.0 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.8.0 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/speakeasy-api/jsonpath v0.6.3 // indirect
	github.com/speakeasy-api/openapi v1.24.0 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
