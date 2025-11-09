// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package petstore

import (
	goerrors "errors"
	"io"
	"net/http"
	"strings"
	gotest "testing"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	testingutil "github.com/go-openapi/runtime/internal/testing"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/security"
	"github.com/go-openapi/runtime/yamlpc"
	"github.com/stretchr/testify/require"
)

const (
	apiPrincipal = "admin"
	apiUser      = "topuser"
	otherUser    = "anyother"
)

// NewAPI registers a stub api for the pet store
func NewAPI(t gotest.TB) (*loads.Document, *untyped.API) {
	spec, err := loads.Analyzed(testingutil.PetStoreJSONMessage, "")
	require.NoError(t, err)
	api := untyped.NewAPI(spec)

	api.RegisterConsumer("application/json", runtime.JSONConsumer())
	api.RegisterProducer("application/json", runtime.JSONProducer())
	api.RegisterConsumer("application/xml", new(stubConsumer))
	api.RegisterProducer("application/xml", new(stubProducer))
	api.RegisterProducer("text/plain", new(stubProducer))
	api.RegisterProducer("text/html", new(stubProducer))
	api.RegisterConsumer("application/x-yaml", yamlpc.YAMLConsumer())
	api.RegisterProducer("application/x-yaml", yamlpc.YAMLProducer())

	api.RegisterAuth("basic", security.BasicAuth(func(username, password string) (any, error) {
		switch {
		case username == apiPrincipal && password == apiPrincipal:
			return apiPrincipal, nil
		case username == apiUser && password == apiUser:
			return apiUser, nil
		case username == otherUser && password == otherUser:
			return otherUser, nil
		default:
			return nil, errors.Unauthenticated("basic")
		}
	}))
	api.RegisterAuth("apiKey", security.APIKeyAuth("X-API-KEY", "header", func(token string) (any, error) {
		if token == "token123" {
			return apiPrincipal, nil
		}
		return nil, errors.Unauthenticated("token")
	}))
	api.RegisterAuthorizer(runtime.AuthorizerFunc(func(r *http.Request, user any) error {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/pets") && user.(string) != apiPrincipal {
			if user.(string) == apiUser {
				return errors.CompositeValidationError(errors.New(errors.InvalidTypeCode, "unauthorized"))
			}

			return goerrors.New("unauthorized")
		}
		return nil
	}))
	api.RegisterOperation("get", "/pets", new(stubOperationHandler))
	api.RegisterOperation("post", "/pets", new(stubOperationHandler))
	api.RegisterOperation("delete", "/pets/{id}", new(stubOperationHandler))
	api.RegisterOperation("get", "/pets/{id}", new(stubOperationHandler))

	api.Models["pet"] = func() any { return new(Pet) }
	api.Models["newPet"] = func() any { return new(Pet) }
	api.Models["tag"] = func() any { return new(Tag) }

	return spec, api
}

// NewRootAPI registers a stub api for the pet store
func NewRootAPI(t gotest.TB) (*loads.Document, *untyped.API) {
	spec, err := loads.Analyzed(testingutil.RootPetStoreJSONMessage, "")
	require.NoError(t, err)
	api := untyped.NewAPI(spec)

	api.RegisterConsumer("application/json", runtime.JSONConsumer())
	api.RegisterProducer("application/json", runtime.JSONProducer())
	api.RegisterConsumer("application/xml", new(stubConsumer))
	api.RegisterProducer("application/xml", new(stubProducer))
	api.RegisterProducer("text/plain", new(stubProducer))
	api.RegisterProducer("text/html", new(stubProducer))
	api.RegisterConsumer("application/x-yaml", yamlpc.YAMLConsumer())
	api.RegisterProducer("application/x-yaml", yamlpc.YAMLProducer())

	api.RegisterAuth("basic", security.BasicAuth(func(username, password string) (any, error) {
		if username == apiPrincipal && password == apiPrincipal {
			return apiPrincipal, nil
		}
		return nil, errors.Unauthenticated("basic")
	}))
	api.RegisterAuth("apiKey", security.APIKeyAuth("X-API-KEY", "header", func(token string) (any, error) {
		if token == "token123" {
			return apiPrincipal, nil
		}
		return nil, errors.Unauthenticated("token")
	}))
	api.RegisterAuthorizer(security.Authorized())
	api.RegisterOperation("get", "/pets", new(stubOperationHandler))
	api.RegisterOperation("post", "/pets", new(stubOperationHandler))
	api.RegisterOperation("delete", "/pets/{id}", new(stubOperationHandler))
	api.RegisterOperation("get", "/pets/{id}", new(stubOperationHandler))

	api.Models["pet"] = func() any { return new(Pet) }
	api.Models["newPet"] = func() any { return new(Pet) }
	api.Models["tag"] = func() any { return new(Tag) }

	return spec, api
}

// Tag the tag model
type Tag struct {
	ID   int64
	Name string
}

// Pet the pet model
type Pet struct {
	ID        int64
	Name      string
	PhotoURLs []string
	Status    string
	Tags      []Tag
}

type stubConsumer struct {
}

func (s *stubConsumer) Consume(_ io.Reader, _ any) error {
	return nil
}

type stubProducer struct {
}

func (s *stubProducer) Produce(_ io.Writer, _ any) error {
	return nil
}

type stubOperationHandler struct {
}

func (s *stubOperationHandler) ParameterModel() any {
	return nil
}

func (s *stubOperationHandler) Handle(_ any) (any, error) {
	return map[string]any{}, nil
}
