package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	require.False(t, DebugEnabled())
}
