// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"bytes"
	"encoding/xml"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

var consProdXML = `<person><name>Somebody</name><id>1</id></person>`

func TestXMLConsumer(t *testing.T) {
	cons := XMLConsumer()
	var data struct {
		XMLName xml.Name `xml:"person"`
		Name    string   `xml:"name"`
		ID      int      `xml:"id"`
	}
	err := cons.Consume(bytes.NewBufferString(consProdXML), &data)
	require.NoError(t, err)
	assert.Equal(t, "Somebody", data.Name)
	assert.Equal(t, 1, data.ID)
}

func TestXMLProducer(t *testing.T) {
	prod := XMLProducer()
	data := struct {
		XMLName xml.Name `xml:"person"`
		Name    string   `xml:"name"`
		ID      int      `xml:"id"`
	}{Name: "Somebody", ID: 1}

	rw := httptest.NewRecorder()
	err := prod.Produce(rw, data)
	require.NoError(t, err)
	assert.Equal(t, consProdXML, rw.Body.String())
}
