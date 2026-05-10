// SPDX-License-Identifier: Apache-2.0

// Command contentnegotiation backs the snippets on the doc-site
// "Content negotiation" page (usage/standalone/content-negotiation.md).
// Each function below is the source of a `{{< code region="..." >}}`
// include; the package as a whole compiles and lints so the snippets
// cannot rot silently.
//
// `go run .` exits immediately (no demos run by default). Pass `serve` to
// run the negotiating HTTP server demo (which blocks on ListenAndServe).
package main

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/go-openapi/runtime/server-middleware/negotiate"
	"github.com/go-openapi/runtime/server-middleware/negotiate/header"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

const (
	readHeaderTimeout = 5 * time.Second
	mediaTypeJSON     = "application/json"
	mediaTypeXML      = "application/xml"
)

// Stubs for the server-wide opt-out snippet. They are package-level so
// the snippet region itself stays focused on the single line that matters.
var (
	spec *loads.Document
	api  *untyped.API
)

func main() {
	pickEncoding(nil, nil)
	ignoreParameters(nil)
	serverWideIgnoreParameters()
	parseAcceptHeader(nil)
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		pickContentType()
	}
}

// --- Snippets -------------------------------------------------------

// snippet:pickContentType

// Pet is the demo resource served by the negotiation handler.
type Pet struct {
	XMLName xml.Name `json:"-"    xml:"pet"`
	Name    string   `json:"name" xml:"name"`
}

func pickContentType() {
	pet := Pet{Name: "Lassie"}
	offers := []string{mediaTypeJSON, mediaTypeXML}

	http.HandleFunc("/pet", func(w http.ResponseWriter, r *http.Request) {
		chosen := negotiate.ContentType(r, offers, mediaTypeJSON)
		w.Header().Set("Content-Type", chosen)

		switch chosen {
		case mediaTypeXML:
			_ = xml.NewEncoder(w).Encode(pet)
		default:
			_ = json.NewEncoder(w).Encode(pet)
		}
	})

	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Fatal(srv.ListenAndServe())
}

// endsnippet:pickContentType

func ignoreParameters(r *http.Request) {
	if r == nil {
		return
	}
	offers := []string{mediaTypeJSON, mediaTypeXML}
	// snippet:ignoreParameters
	chosen := negotiate.ContentType(r, offers, "",
		negotiate.WithIgnoreParameters(true),
	)
	// endsnippet:ignoreParameters

	use(chosen)
}

func serverWideIgnoreParameters() {
	// snippet:serverWideIgnoreParameters
	ctx := middleware.NewContext(spec, api, nil).SetIgnoreParameters(true)
	// endsnippet:serverWideIgnoreParameters

	use(ctx)
}

func pickEncoding(w http.ResponseWriter, r *http.Request) {
	if r == nil || w == nil {
		return
	}
	// snippet:pickEncoding
	chosen := negotiate.ContentEncoding(r, []string{"gzip", "deflate"})
	if chosen != "" {
		w.Header().Set("Content-Encoding", chosen)
	}
	// endsnippet:pickEncoding

	use(chosen)
}

func parseAcceptHeader(r *http.Request) {
	if r == nil {
		return
	}
	// snippet:parseAcceptHeader
	specs := header.ParseAccept(r.Header, "Accept")
	for _, s := range specs {
		// s.Value, s.Q, s.Params
		use(s)
	}
	// endsnippet:parseAcceptHeader
}
