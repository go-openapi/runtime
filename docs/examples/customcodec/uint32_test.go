// SPDX-License-Identifier: Apache-2.0

package customcodec

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

func TestUint32Consumer_DecodesBigEndian(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.BigEndian, uint32(0xCAFEBABE)))

	var got uint32
	require.NoError(t, Uint32Consumer().Consume(&buf, &got))
	require.Equal(t, uint32(0xCAFEBABE), got)
}

func TestUint32Consumer_RejectsWrongTargetType(t *testing.T) {
	var notAUint32 int
	err := Uint32Consumer().Consume(strings.NewReader("\x00\x00\x00\x01"), &notAUint32)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not *uint32")
}
