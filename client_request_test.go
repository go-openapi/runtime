// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestRequestWriterFunc(t *testing.T) {
	hand := ClientRequestWriterFunc(func(r ClientRequest, _ strfmt.Registry) error {
		_ = r.SetHeaderParam("blah", "blahblah")
		_ = r.SetBodyParam(struct{ Name string }{"Adriana"})
		return nil
	})

	tr := new(TestClientRequest)
	_ = hand.WriteToRequest(tr, nil)
	assert.Equal(t, "blahblah", tr.Headers.Get("blah"))
	assert.Equal(t, "Adriana", tr.Body.(struct{ Name string }).Name)
}
