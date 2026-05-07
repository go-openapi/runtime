// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package docui

// minimal valid swagger 2.0 document, sufficient for round-tripping bytes
// without dragging in the petstore fixture (which lives in an internal/
// package of the parent runtime module and is not importable from here).
var testSpec = []byte(`{"swagger":"2.0","info":{"title":"Test","version":"1.0.0"},"paths":{}}`)

// badTemplate references a field that does not exist on the [options]
// struct. Parsing succeeds but execution fails — exercising the
// template.Execute panic branch in every UI handler.
const badTemplate = `<!DOCTYPE html>
<html>
	spec-url='{{ .Unknown }}'
</html>
`

// malformedTemplate fails at parse time (open action with no close).
const malformedTemplate = `<!DOCTYPE html>
<html>
  <head>
		spec-url='{{ .Spec
</html>
`
