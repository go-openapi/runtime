package middleware

import (
	"bytes"
	stdcontext "context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/runtime/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customLogger struct {
	logger.StandardLogger
	lg *log.Logger
}

func (l customLogger) Debugf(format string, args ...interface{}) {
	l.lg.Printf(format, args...)
}

func TestDebugMode(t *testing.T) {
	t.Run("with normal mode", func(t *testing.T) {
		t.Setenv("DEBUG", "")

		logFunc := debugLogfFunc(nil)
		require.NotNil(t, logFunc)
	})

	t.Run("with debug mode", func(t *testing.T) {
		t.Setenv("DEBUG", "true")

		t.Run("debugLogFunc with nil logger yields standard logger", func(t *testing.T) {
			logFunc := debugLogfFunc(nil)
			require.NotNil(t, logFunc)
		})
		t.Run("debugLogFunc with custom logger", func(t *testing.T) {
			var capture bytes.Buffer
			logger := customLogger{lg: log.New(&capture, "test", log.Lshortfile)}
			logFunc := debugLogfFunc(logger)
			require.NotNil(t, logFunc)

			logFunc("debug")
			assert.NotEmpty(t, capture.String())
		})
	})
}

func TestDebugRouterOptions(t *testing.T) {
	t.Run("with normal mode", func(t *testing.T) {
		t.Setenv("DEBUG", "")

		t.Run("should capture debug from context & router", func(t *testing.T) {
			var capture bytes.Buffer
			logger := customLogger{lg: log.New(&capture, "test", log.Lshortfile)}

			t.Run("run some activiy", doCheckWithContext(logger))
			assert.Empty(t, capture.String())
		})

		t.Run("should capture debug from standalone DefaultRouter", func(t *testing.T) {
			var capture bytes.Buffer
			logger := customLogger{lg: log.New(&capture, "test", log.Lshortfile)}

			t.Run("run some activiy", doCheckWithDefaultRouter(logger))
			assert.Empty(t, capture.String())
		})
	})

	t.Run("with debug mode", func(t *testing.T) {
		t.Setenv("DEBUG", "1")

		t.Run("should capture debug from context & router", func(t *testing.T) {
			var capture bytes.Buffer
			logger := customLogger{lg: log.New(&capture, "test", log.Lshortfile)}

			t.Run("run some activiy", doCheckWithContext(logger))
			assert.NotEmpty(t, capture.String())
		})

		t.Run("should capture debug from standalone DefaultRouter", func(t *testing.T) {
			var capture bytes.Buffer
			logger := customLogger{lg: log.New(&capture, "test", log.Lshortfile)}

			t.Run("run some activiy", doCheckWithDefaultRouter(logger))
			assert.NotEmpty(t, capture.String())
		})
	})
}

func doCheckWithContext(logger logger.Logger) func(*testing.T) {
	return func(t *testing.T) {
		spec, api := petstore.NewAPI(t)
		context := NewContext(spec, api, nil)
		context.SetLogger(logger)
		mw := NewRouter(context, http.HandlerFunc(terminator))

		recorder := httptest.NewRecorder()
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
		require.NoError(t, err)
		mw.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusOK, recorder.Code)
	}
}

func doCheckWithDefaultRouter(lg logger.Logger) func(*testing.T) {
	return func(t *testing.T) {
		spec, api := petstore.NewAPI(t)
		context := NewContext(spec, api, nil)
		context.SetLogger(lg)
		router := DefaultRouter(
			spec,
			newRoutableUntypedAPI(spec, api, new(Context)),
			WithDefaultRouterLogger(lg))

		_ = router.OtherMethods("post", "/api/pets/{id}")
	}
}
