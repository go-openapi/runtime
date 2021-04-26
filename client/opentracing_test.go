package client

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-openapi/runtime"
)

type tres struct {
}

func (r tres) Code() int {
	return 490
}
func (r tres) Message() string {
	return "the message"
}
func (r tres) GetHeader(_ string) string {
	return "the header"
}
func (r tres) Body() io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBufferString("the content"))
}

type mockRuntime struct {
	req runtime.TestClientRequest
}

func (m *mockRuntime) Submit(operation *runtime.ClientOperation) (interface{}, error) {
	_ = operation.AuthInfo.AuthenticateRequest(&m.req, nil)
	_, _ = operation.Reader.ReadResponse(&tres{}, nil)
	return nil, nil
}

func testOperation(ctx context.Context) *runtime.ClientOperation {
	return &runtime.ClientOperation{
		ID:                 "getCluster",
		Method:             "GET",
		PathPattern:        "/kubernetes-clusters/{cluster_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Reader: runtime.ClientResponseReaderFunc(func(runtime.ClientResponse, runtime.Consumer) (interface{}, error) {
			return nil, nil
		}),
		AuthInfo: PassThroughAuth,
		Context:  ctx,
	}
}

func Test_TracingRuntime_submit(t *testing.T) {
	t.Parallel()
	tracer := mocktracer.New()
	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), tracer, "op")
	testSubmit(t, testOperation(ctx), tracer)
}

func Test_TracingRuntime_submit_nullAuthInfo(t *testing.T) {
	t.Parallel()
	tracer := mocktracer.New()
	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), tracer, "op")
	operation := testOperation(ctx)
	operation.AuthInfo = nil
	testSubmit(t, operation, tracer)
}

func testSubmit(t *testing.T, operation *runtime.ClientOperation, tracer *mocktracer.MockTracer) {

	header := map[string][]string{}
	r := newOpenTracingTransport(&mockRuntime{runtime.TestClientRequest{Headers: header}},
		"remote_host",
		[]opentracing.StartSpanOption{opentracing.Tag{
			Key:   string(ext.PeerService),
			Value: "service",
		}})

	_, err := r.Submit(operation)
	require.NoError(t, err)

	if assert.Len(t, tracer.FinishedSpans(), 1) {
		span := tracer.FinishedSpans()[0]
		assert.Equal(t, "get_cluster", span.OperationName)
		assert.Equal(t, map[string]interface{}{
			"component":        "go-openapi",
			"http.method":      "GET",
			"http.path":        "/kubernetes-clusters/{cluster_id}",
			"http.status_code": uint16(490),
			"peer.hostname":    "remote_host",
			"peer.service":     "service",
			"span.kind":        ext.SpanKindRPCClientEnum,
			"error":            true,
		}, span.Tags())
	}
}

func Test_injectSpanContext(t *testing.T) {
	t.Parallel()
	tracer := mocktracer.New()
	_, ctx := opentracing.StartSpanFromContextWithTracer(context.Background(), tracer, "op")
	header := map[string][]string{}
	createClientSpan(testOperation(ctx), header, "", nil)

	// values are random - just check that something was injected
	assert.Len(t, header, 3)
}
