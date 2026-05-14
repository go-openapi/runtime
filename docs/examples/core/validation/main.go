// SPDX-License-Identifier: Apache-2.0

// Command validation backs the snippets on the doc-site
// "Validation hooks" page. Each type below is the source of a
// `{{< code region="..." >}}` include; the package as a whole
// compiles and lints so the snippets cannot rot silently.
//
// `go run .` exercises the validators against canned inputs.
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/go-openapi/strfmt"
)

// --- Stubs (excluded from rendered snippets) ------------------------

// userKey is the context-key type used by the fake reqUser helper.
type userKey struct{}

// reqUser pretends to extract the authenticated user from a request
// context. Real code would read from a request-scoped middleware value.
func reqUser(ctx context.Context) string {
	if v, ok := ctx.Value(userKey{}).(string); ok {
		return v
	}
	return ""
}

func main() {
	dateRangeValidation()
	contextValidation()
}

func dateRangeValidation() {
	from := strfmt.Date(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	to := strfmt.Date(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	good := DateRange{From: from, To: to}
	bad := DateRange{From: to, To: from}

	log.Println("good DateRange:", good.Validate(strfmt.Default))
	log.Println("bad DateRange:", bad.Validate(strfmt.Default))
}

func contextValidation() {
	anon := context.Background()
	authed := context.WithValue(context.Background(), userKey{}, "alice")

	req := MyRequest{OnBehalfOf: "bob"}
	log.Println("anonymous on_behalf_of:", req.ContextValidate(anon, strfmt.Default))
	log.Println("authenticated on_behalf_of:", req.ContextValidate(authed, strfmt.Default))
}

// --- Snippets -------------------------------------------------------

// snippet:dateRangeValidate

// DateRange illustrates a cross-field invariant on a hand-written type.
type DateRange struct {
	From strfmt.Date `json:"from"`
	To   strfmt.Date `json:"to"`
}

// Validate enforces that To is not before From. The strfmt.Registry
// argument is unused here because the rule does not involve any
// registered string format.
func (d DateRange) Validate(_ strfmt.Registry) error {
	if time.Time(d.To).Before(time.Time(d.From)) {
		return errors.New("DateRange.to must not be before DateRange.from")
	}
	return nil
}

// endsnippet:dateRangeValidate

// MyRequest is a hand-written request type that demonstrates a
// context-aware validation rule.
type MyRequest struct {
	OnBehalfOf string `json:"onBehalfOf"`
}

// snippet:contextValidate

// ContextValidate enforces that on_behalf_of is only set when the
// request context carries an authenticated user.
func (r MyRequest) ContextValidate(ctx context.Context, _ strfmt.Registry) error {
	if reqUser(ctx) == "" && r.OnBehalfOf != "" {
		return errors.New("on_behalf_of is only valid when authenticated")
	}
	return nil
}

// endsnippet:contextValidate
