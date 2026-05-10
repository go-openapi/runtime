// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package main shows how to add transparent response compression
// (gzip, brotli, etc.) to a go-openapi server by wrapping the
// http.Handler returned by middleware.Serve with the CAFxX
// httpcompression adapter.
//
// The runtime does not ship compression itself; this example is a
// recipe for composing the existing ecosystem with the go-openapi
// pipeline. See server-middleware/negotiate.ContentEncoding (now
// deprecated) for context.
//
// Run:
//
//	go run .
//	curl -sH 'Accept-Encoding: gzip' -i http://localhost:8080/api/greeting | head
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	httpcompression "github.com/CAFxX/httpcompression"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
)

const (
	// greetingPayloadKeys keeps the response body comfortably above
	// the compression middleware's MinSize threshold so the example
	// actually exercises the compressor instead of falling through.
	greetingPayloadKeys = 32

	listenAddress     = ":8080"
	readHeaderTimeout = 5 * time.Second
)

const swaggerSpec = `{
  "swagger": "2.0",
  "info": {"title": "Compression Demo", "version": "1.0"},
  "basePath": "/api",
  "consumes": ["application/json"],
  "produces": ["application/json"],
  "paths": {
    "/greeting": {
      "get": {
        "operationId": "greeting",
        "responses": {
          "200": {
            "description": "a greeting payload large enough to be worth compressing",
            "schema": {"type": "object"}
          }
        }
      }
    }
  }
}`

// greeting returns a payload large enough to clear the compression
// middleware's default minimum-size threshold (CAFxX defaults to
// DefaultMinSize = 200 bytes — payloads below that are left
// uncompressed because the wire overhead would outweigh the win).
var greeting = runtime.OperationHandlerFunc(func(_ any) (any, error) {
	greetings := make(map[string]string, greetingPayloadKeys)
	for i := range greetingPayloadKeys {
		greetings[encodeKey(i)] = "Hello from go-openapi! Compression makes this body smaller on the wire."
	}
	return greetings, nil
})

func encodeKey(i int) string {
	return "greeting_" + string(rune('a'+i%26))
}

func newAPI() (http.Handler, error) {
	spec, err := loads.Analyzed(json.RawMessage(swaggerSpec), "")
	if err != nil {
		return nil, err
	}
	api := untyped.NewAPI(spec)
	api.RegisterOperation("get", "/greeting", greeting)
	return middleware.Serve(spec, api), nil
}

func main() {
	apiHandler, err := newAPI()
	if err != nil {
		log.Fatalf("build api: %v", err)
	}

	// DefaultAdapter wires gzip + brotli encoders with sane defaults:
	// content-type allowlist, minimum-size threshold, Vary and
	// Content-Length handling, ETag suffixing for cacheability.
	//
	// Use Adapter (without "Default") for explicit codec + threshold
	// control:
	//
	//   compress, err := httpcompression.Adapter(
	//       httpcompression.GzipCompressionLevel(6),
	//       httpcompression.BrotliCompressionLevel(4),
	//       httpcompression.MinSize(512),
	//       httpcompression.ContentTypes([]string{"application/json"}, false),
	//   )
	// snippet:compressionWiring
	compress, err := httpcompression.DefaultAdapter()
	if err != nil {
		log.Fatalf("compression adapter: %v", err)
	}

	// Wrap the go-openapi handler. The order matters:
	//   - the compressor must be OUTSIDE the api pipeline so it sees
	//     the final response bytes;
	//   - any TLS / auth / rate-limiting middleware typically wraps
	//     the compressor (i.e. compressor sits between application
	//     code and transport-level middleware).
	mux := http.NewServeMux()
	mux.Handle("/", compress(apiHandler))
	// endsnippet:compressionWiring

	log.Printf("listening on %s", listenAddress)
	srv := &http.Server{
		Addr:              listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Fatal(srv.ListenAndServe())
}
