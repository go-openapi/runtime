package client

import (
	"bytes"
	"context"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

type mockRuntime struct {
	req runtime.TestClientRequest
}

func (m *mockRuntime) Submit(operation *runtime.ClientOperation) (any, error) {
	_ = operation.Params.WriteToRequest(&m.req, nil)
	_, _ = operation.Reader.ReadResponse(&tres{}, nil)
	return map[string]any{}, nil
}

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
func (r tres) GetHeaders(_ string) []string {
	return []string{"the headers", "the headers2"}
}
func (r tres) Body() io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString("the content"))
}

func testOperation(ctx context.Context) *runtime.ClientOperation {
	return &runtime.ClientOperation{
		ID:                 "getCluster",
		Method:             "GET",
		PathPattern:        "/kubernetes-clusters/{cluster_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Reader: runtime.ClientResponseReaderFunc(func(runtime.ClientResponse, runtime.Consumer) (any, error) {
			return nil, nil
		}),
		Params: runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
			return nil
		}),
		AuthInfo: PassThroughAuth,
		Context:  ctx,
	}
}
