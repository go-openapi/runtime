package flagext

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalBytesize(t *testing.T) {
	v, err := ByteSize(1024).MarshalFlag()
	require.NoError(t, err)
	assert.Equal(t, "1.024kB", v)
}

func TestStringBytesize(t *testing.T) {
	v := ByteSize(2048).String()
	assert.Equal(t, "2.048kB", v)
}

func TestUnmarshalBytesize(t *testing.T) {
	var b ByteSize
	err := b.UnmarshalFlag("notASize")
	require.Error(t, err)

	err = b.UnmarshalFlag("1MB")
	require.NoError(t, err)
	assert.Equal(t, ByteSize(1000000), b)
}

func TestSetBytesize(t *testing.T) {
	var b ByteSize
	err := b.Set("notASize")
	require.Error(t, err)

	err = b.Set("2MB")
	require.NoError(t, err)
	assert.Equal(t, ByteSize(2000000), b)
}

func TestTypeBytesize(t *testing.T) {
	var b ByteSize
	assert.Equal(t, "byte-size", b.Type())
}
