// SPDX-License-Identifier: Apache-2.0

// Command mediatypes backs the snippets on the doc-site
// "Media types" page. Each function below is the source of a
// `{{< code region="..." >}}` include; the package as a whole
// compiles and lints so the snippets cannot rot silently.
//
// `go run .` exercises the demo parse call.
package main

import (
	"errors"
	"log"

	"github.com/go-openapi/runtime/server-middleware/mediatype"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// use silences "declared and not used" diagnostics for snippet-local
// values that exist only to make the demo compile.
func use(_ ...any) {}

func main() {
	parseMediaType()
}

// --- Snippets -------------------------------------------------------

func parseMediaType() {
	// snippet:parseMediaType
	mt, err := mediatype.Parse("application/json;charset=utf-8;q=0.8")
	// mt.Type    = "application"
	// mt.Subtype = "json"
	// mt.Params  = {"charset": "utf-8"}
	// mt.Q       = 0.8

	if errors.Is(err, mediatype.ErrMalformed) {
		// ↳ 400 Bad Request territory
		log.Println("malformed media type")
	}
	// endsnippet:parseMediaType

	use(mt, err)
}
