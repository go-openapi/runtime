package testing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvalidJSON(t *testing.T) {
	require.NotEmpty(t, InvalidJSONMessage)
}
