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

package yamlpc

import (
	"bytes"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var consProdYAML = "name: Somebody\nid: 1\n"

func TestYAMLConsumer(t *testing.T) {
	cons := YAMLConsumer()
	var data struct {
		Name string
		ID   int
	}
	err := cons.Consume(bytes.NewBufferString(consProdYAML), &data)
	require.NoError(t, err)
	assert.Equal(t, "Somebody", data.Name)
	assert.Equal(t, 1, data.ID)
}

func TestYAMLProducer(t *testing.T) {
	prod := YAMLProducer()
	data := struct {
		Name string `yaml:"name"`
		ID   int    `yaml:"id"`
	}{Name: "Somebody", ID: 1}

	rw := httptest.NewRecorder()
	err := prod.Produce(rw, data)
	require.NoError(t, err)
	assert.Equal(t, consProdYAML, rw.Body.String())
}

type failReaderWriter struct {
}

func (f *failReaderWriter) Read(_ []byte) (n int, err error) {
	return 0, errors.New("expected")
}

func (f *failReaderWriter) Write(_ []byte) (n int, err error) {
	return 0, errors.New("expected")
}

func TestFailYAMLWriter(t *testing.T) {
	prod := YAMLProducer()
	require.Error(t, prod.Produce(&failReaderWriter{}, nil))
}

func TestFailYAMLReader(t *testing.T) {
	cons := YAMLConsumer()
	require.Error(t, cons.Consume(&failReaderWriter{}, nil))
}

func TestYAMLConsumerObject(t *testing.T) {
	const yamlDoc = `
---
name: fred
id: 123
attributes:
  height: 12.3
  weight: 45
  list:
    - a
    - b
`
	cons := YAMLConsumer()
	var data struct {
		Name       string
		ID         int
		Attributes struct {
			Height float64
			Weight uint64
			List   []string
		}
	}
	require.NoError(t,
		cons.Consume(bytes.NewBufferString(yamlDoc), &data),
	)

	assert.Equal(t, "fred", data.Name)
	assert.Equal(t, 123, data.ID)
	assert.InDelta(t, 12.3, data.Attributes.Height, 1e-9)
	assert.Equal(t, uint64(45), data.Attributes.Weight)
	assert.Len(t, data.Attributes.List, 2)
	assert.Equal(t, "a", data.Attributes.List[0])
}
