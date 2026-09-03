// Module github.com/go-openapi/runtime/docs/examples hosts runnable code
// samples referenced from the documentation site. It is intentionally kept
// separate from the root module so example dependencies do not leak into
// runtime consumers.
module github.com/go-openapi/runtime/docs/examples

go 1.26.0

require (
	github.com/CAFxX/httpcompression v0.0.9
	github.com/go-openapi/analysis v1.0.0
	github.com/go-openapi/errors v0.22.8
	github.com/go-openapi/loads v0.25.2
	github.com/go-openapi/runtime v0.33.1
	github.com/go-openapi/runtime/server-middleware v0.33.1
	github.com/go-openapi/strfmt v0.27.1
	github.com/go-openapi/testify/v2 v2.7.0
	github.com/justinas/alice v1.2.0
	go.opentelemetry.io/otel v1.46.0
	go.opentelemetry.io/otel/trace v1.46.0
)

require (
	github.com/andybalholm/brotli v1.2.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/go-openapi/jsonreference v1.0.1 // indirect
	github.com/go-openapi/spec v1.0.0 // indirect
	github.com/go-openapi/swag/conv v0.29.1 // indirect
	github.com/go-openapi/swag/fileutils v0.29.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.29.1 // indirect
	github.com/go-openapi/swag/loading v0.29.1 // indirect
	github.com/go-openapi/swag/mangling v0.29.1 // indirect
	github.com/go-openapi/swag/pools v0.29.1 // indirect
	github.com/go-openapi/swag/stringutils v0.29.1 // indirect
	github.com/go-openapi/swag/typeutils v0.29.1 // indirect
	github.com/go-openapi/swag/yamlutils v0.29.1 // indirect
	github.com/go-openapi/validate v1.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace (
	github.com/go-openapi/runtime => ../..
	github.com/go-openapi/runtime/server-middleware => ../../server-middleware
)
