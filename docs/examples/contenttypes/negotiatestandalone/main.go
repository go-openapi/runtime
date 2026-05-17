// SPDX-License-Identifier: Apache-2.0

// Command negotiatestandalone backs the snippets on the doc-site
// "Negotiation in plain net/http" page. Each function below is the source
// of a `{{< code region="..." >}}` include; the package as a whole
// compiles and lints so the snippets cannot rot silently.
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

	"github.com/go-openapi/runtime/server-middleware/negotiate"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

const readHeaderTimeout = 5 * time.Second

func main() {
	ignoreParameters(nil)
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		pickContentType()
	}
}

// --- Snippets -------------------------------------------------------

// snippet:pickContentType

const mediaTypeXML = "application/xml"

// Pet is the demo resource served by the negotiation handler.
type Pet struct {
	XMLName xml.Name `json:"-"    xml:"pet"`
	Name    string   `json:"name" xml:"name"`
}

func pickContentType() {
	pet := Pet{Name: "Lassie"}
	offers := []string{"application/json", mediaTypeXML}

	http.HandleFunc("/pet", func(w http.ResponseWriter, r *http.Request) {
		chosen := negotiate.ContentType(r, offers, "application/json")
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
	offers := []string{"application/json", mediaTypeXML}
	// snippet:ignoreParameters
	chosen := negotiate.ContentType(r, offers, "",
		negotiate.WithIgnoreParameters(true),
	)
	// endsnippet:ignoreParameters

	use(chosen)
}
