package header

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeader(t *testing.T) {
	hdr := http.Header{
		"x-test": []string{"value"},
	}
	clone := Copy(hdr)
	require.Len(t, clone, len(hdr))
}
