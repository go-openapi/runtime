package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetOK(t *testing.T) {
	m := make(map[string][]string)
	m["key1"] = []string{"value1"}
	m["key2"] = []string{}
	values := Values(m)

	v, hasKey, hasValue := values.GetOK("key1")
	require.Equal(t, []string{"value1"}, v)
	require.True(t, hasKey)
	require.True(t, hasValue)

	v, hasKey, hasValue = values.GetOK("key2")
	require.Equal(t, []string{}, v)
	require.True(t, hasKey)
	require.False(t, hasValue)

	v, hasKey, hasValue = values.GetOK("key3")
	require.Nil(t, v)
	require.False(t, hasKey)
	require.False(t, hasValue)
}
