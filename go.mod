module github.com/go-openapi/runtime

require (
	github.com/docker/go-units v0.5.0
	github.com/go-openapi/analysis v0.25.1
	github.com/go-openapi/errors v0.22.7
	github.com/go-openapi/loads v0.23.3
	github.com/go-openapi/runtime/server-middleware v0.30.0
	github.com/go-openapi/spec v0.22.4
	github.com/go-openapi/strfmt v0.26.2
	github.com/go-openapi/swag/conv v0.26.0
	github.com/go-openapi/swag/fileutils v0.26.0
	github.com/go-openapi/swag/jsonutils v0.26.0
	github.com/go-openapi/swag/stringutils v0.26.0
	github.com/go-openapi/swag/typeutils v0.26.0
	github.com/go-openapi/testify/enable/yaml/v2 v2.5.1
	github.com/go-openapi/testify/v2 v2.5.1
	github.com/go-openapi/validate v0.25.2
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	go.yaml.in/yaml/v3 v3.0.4
	golang.org/x/sync v0.20.0
)

replace github.com/go-openapi/runtime/server-middleware => ./server-middleware

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/jsonreference v0.21.5 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/go-openapi/swag/loading v0.26.0 // indirect
	github.com/go-openapi/swag/mangling v0.26.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.26.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)

go 1.25.0
