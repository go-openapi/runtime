// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yamlpc

import (
	"bytes"
	"errors"
	"net/http/httptest"
	"testing"

	_ "github.com/go-openapi/testify/enable/yaml/v2"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
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
	assert.YAMLEq(t, consProdYAML, rw.Body.String())
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
