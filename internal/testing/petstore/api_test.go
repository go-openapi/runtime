package petstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	doc, api := NewAPI(t)

	require.NotNil(t, doc)
	require.NotNil(t, api)
}
