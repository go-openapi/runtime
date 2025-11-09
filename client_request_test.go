// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestRequestWriterFunc(t *testing.T) {
	hand := ClientRequestWriterFunc(func(r ClientRequest, _ strfmt.Registry) error {
		_ = r.SetHeaderParam("Blah", "blahblah")
		_ = r.SetBodyParam(struct{ Name string }{"Adriana"})
		return nil
	})

	tr := new(TestClientRequest)
	_ = hand.WriteToRequest(tr, nil)
	assert.Equal(t, "blahblah", tr.Headers.Get("Blah"))
	assert.Equal(t, "Adriana", tr.Body.(struct{ Name string }).Name)
}
