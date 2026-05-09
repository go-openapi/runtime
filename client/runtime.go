// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client/internal/request"
	"github.com/go-openapi/runtime/logger"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/yamlpc"
	"github.com/go-openapi/strfmt"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

// DefaultTimeout the default request timeout.
var DefaultTimeout = 30 * time.Second

// Runtime represents an API client that uses the transport
// to make [http] requests based on a swagger specification.
type Runtime struct {
	DefaultMediaType      string
	DefaultAuthentication runtime.ClientAuthInfoWriter
	Consumers             map[string]runtime.Consumer
	Producers             map[string]runtime.Producer

	Transport http.RoundTripper
	Jar       http.CookieJar
	// Spec      *spec.Document
	Host     string
	BasePath string
	Formats  strfmt.Registry
	Context  context.Context //nolint:containedctx  // we precisely want this type to contain the request context

	Debug  bool
	logger logger.Logger

	clientOnce *sync.Once
	client     *http.Client
	schemes    []string
	response   ClientResponseFunc
}

// New creates a new default runtime for a swagger api runtime.Client.
func New(host, basePath string, schemes []string) *Runtime {
	var rt Runtime
	rt.DefaultMediaType = runtime.JSONMime

	// Enhancement proposal: https://github.com/go-openapi/runtime/issues/385
	rt.Consumers = map[string]runtime.Consumer{
		runtime.YAMLMime:    yamlpc.YAMLConsumer(),
		runtime.JSONMime:    runtime.JSONConsumer(),
		runtime.XMLMime:     runtime.XMLConsumer(),
		runtime.TextMime:    runtime.TextConsumer(),
		runtime.HTMLMime:    runtime.TextConsumer(),
		runtime.CSVMime:     runtime.CSVConsumer(),
		runtime.DefaultMime: runtime.ByteStreamConsumer(),
	}
	rt.Producers = map[string]runtime.Producer{
		runtime.YAMLMime:    yamlpc.YAMLProducer(),
		runtime.JSONMime:    runtime.JSONProducer(),
		runtime.XMLMime:     runtime.XMLProducer(),
		runtime.TextMime:    runtime.TextProducer(),
		runtime.HTMLMime:    runtime.TextProducer(),
		runtime.CSVMime:     runtime.CSVProducer(),
		runtime.DefaultMime: runtime.ByteStreamProducer(),
	}
	rt.Transport = http.DefaultTransport
	rt.Jar = nil
	rt.Host = host
	rt.BasePath = basePath
	rt.Context = context.Background()
	rt.clientOnce = new(sync.Once)
	if !strings.HasPrefix(rt.BasePath, "/") {
		rt.BasePath = "/" + rt.BasePath
	}

	rt.Debug = logger.DebugEnabled()
	rt.logger = logger.StandardLogger{}
	rt.response = newResponse

	if len(schemes) > 0 {
		rt.schemes = schemes
	}
	return &rt
}

// NewWithClient allows you to create a new transport with a configured [http.Client].
func NewWithClient(host, basePath string, schemes []string, client *http.Client) *Runtime {
	rt := New(host, basePath, schemes)
	if client != nil {
		rt.clientOnce.Do(func() {
			rt.client = client
		})
	}
	return rt
}

// EnableConnectionReuse drains the remaining body from a response
// so that go will reuse the TCP connections.
//
// This is not enabled by default because there are servers where
// the response never gets closed and that would make the code hang forever.
// So instead it's provided as a [http] client [middleware] that can be used to override
// any request.
func (r *Runtime) EnableConnectionReuse() {
	if r.client == nil {
		r.Transport = KeepAliveTransport(
			transportOrDefault(r.Transport, http.DefaultTransport),
		)
		return
	}

	r.client.Transport = KeepAliveTransport(
		transportOrDefault(r.client.Transport,
			transportOrDefault(r.Transport, http.DefaultTransport),
		),
	)
}

func (r *Runtime) CreateHttpRequest(operation *runtime.ClientOperation) (req *http.Request, err error) { //nolint:revive
	_, req, err = r.createHttpRequest(operation)
	return
}

// Submit a request and when there is a body on success it will turn that into the result
// all other things are turned into an api error for swagger which retains the status code.
func (r *Runtime) Submit(operation *runtime.ClientOperation) (any, error) {
	_, readResponse, _ := operation.Params, operation.Reader, operation.AuthInfo

	request, req, err := r.createHttpRequest(operation)
	if err != nil {
		return nil, err
	}

	r.clientOnce.Do(func() {
		r.client = &http.Client{
			Transport: r.Transport,
			Jar:       r.Jar,
		}
	})

	if r.Debug {
		b, err2 := httputil.DumpRequestOut(req, true)
		if err2 != nil {
			return nil, err2
		}
		r.logger.Debugf("%s\n", string(b))
	}

	var parentCtx context.Context
	switch {
	case operation.Context != nil:
		parentCtx = operation.Context
	case r.Context != nil:
		parentCtx = r.Context
	default:
		parentCtx = context.Background()
	}

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	timeout := request.GetTimeout()
	if timeout == 0 {
		// There may be a deadline in the context passed to the operation.
		// Otherwise, there is no timeout set.
		ctx, cancel = context.WithCancel(parentCtx)
	} else {
		// Sets the timeout passed from request params (by default runtime.DefaultTimeout).
		// If there is already a deadline in the parent context, the shortest will
		// apply.
		ctx, cancel = context.WithTimeout(parentCtx, timeout)
	}
	defer cancel()

	var client *http.Client
	if operation.Client != nil {
		client = operation.Client
	} else {
		client = r.client
	}
	req = req.WithContext(ctx)
	res, err := client.Do(req) // make requests, by default follows 10 redirects before failing
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ct := res.Header.Get(runtime.HeaderContentType)
	if ct == "" { // this should really never occur
		ct = r.DefaultMediaType
	}

	if r.Debug {
		printBody := true
		if ct == runtime.DefaultMime {
			printBody = false // Spare the terminal from a binary blob.
		}
		b, err2 := httputil.DumpResponse(res, printBody)
		if err2 != nil {
			return nil, err2
		}
		r.logger.Debugf("%s\n", string(b))
	}

	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return nil, fmt.Errorf("parse content type: %s", err)
	}

	cons, ok := r.Consumers[mt]
	if !ok {
		if cons, ok = r.Consumers["*/*"]; !ok {
			// scream about not knowing what to do
			return nil, fmt.Errorf("no consumer: %q", ct)
		}
	}
	return readResponse.ReadResponse(r.response(res), cons)
}

// SetDebug changes the debug flag.
// It ensures that client and middlewares have the set debug level.
func (r *Runtime) SetDebug(debug bool) {
	r.Debug = debug
	middleware.Debug = debug
}

// SetLogger changes the logger stream.
// It ensures that client and middlewares use the same logger.
func (r *Runtime) SetLogger(logger logger.Logger) {
	r.logger = logger
	middleware.Logger = logger
}

type ClientResponseFunc = func(*http.Response) runtime.ClientResponse //nolint:revive

// SetResponseReader changes the response reader implementation.
func (r *Runtime) SetResponseReader(f ClientResponseFunc) {
	if f == nil {
		return
	}
	r.response = f
}

func (r *Runtime) pickScheme(schemes []string) string {
	if v := r.selectScheme(r.schemes); v != "" {
		return v
	}
	if v := r.selectScheme(schemes); v != "" {
		return v
	}
	return schemeHTTP
}

func (r *Runtime) selectScheme(schemes []string) string {
	schLen := len(schemes)
	if schLen == 0 {
		return ""
	}

	scheme := schemes[0]
	// prefer https, but skip when not possible
	if scheme != schemeHTTPS && schLen > 1 {
		for _, sch := range schemes {
			if sch == schemeHTTPS {
				scheme = sch
				break
			}
		}
	}
	return scheme
}

func transportOrDefault(left, right http.RoundTripper) http.RoundTripper {
	if left == nil {
		return right
	}
	return left
}

// takes a client operation and creates equivalent http.Request.
func (r *Runtime) createHttpRequest(operation *runtime.ClientOperation) (*request.Request, *http.Request, error) { //nolint:revive
	params, _, auth := operation.Params, operation.Reader, operation.AuthInfo

	req := request.New(operation.Method, operation.PathPattern, params)
	_ = req.SetTimeout(DefaultTimeout)
	req.SetConsumes(operation.ConsumesMediaTypes)

	accept := make([]string, 0, len(operation.ProducesMediaTypes))
	accept = append(accept, operation.ProducesMediaTypes...)
	if err := req.SetHeaderParam(runtime.HeaderAccept, accept...); err != nil {
		return nil, nil, err
	}

	if auth == nil && r.DefaultAuthentication != nil {
		auth = runtime.ClientAuthInfoWriterFunc(func(req runtime.ClientRequest, reg strfmt.Registry) error {
			if req.GetHeaderParams().Get(runtime.HeaderAuthorization) != "" {
				return nil
			}
			return r.DefaultAuthentication.AuthenticateRequest(req, reg)
		})
	}

	cmt := pickConsumesMediaType(operation.ConsumesMediaTypes, r.Producers, r.DefaultMediaType)
	if _, ok := r.Producers[cmt]; !ok && cmt != runtime.MultipartFormMime && cmt != runtime.URLencodedFormMime {
		return nil, nil, fmt.Errorf("none of producers: %v registered. try %s", r.Producers, cmt)
	}

	httpReq, err := req.BuildHTTP(cmt, r.BasePath, r.Producers, r.Formats, auth)
	if err != nil {
		return nil, nil, err
	}

	httpReq.URL.Scheme = r.pickScheme(operation.Schemes)
	httpReq.URL.Host = r.Host
	httpReq.Host = r.Host

	return req, httpReq, nil
}

// pickConsumesMediaType selects which Content-Type the client will send.
//
// Selection rules, in priority order:
//
//  1. multipart/form-data if any consumes entry advertises it (it streams
//     and preserves per-file Content-Type, regardless of codegen ordering;
//     resolves issue #286);
//  2. the first non-empty entry whose mime is either structural
//     (multipart/form-data or application/x-www-form-urlencoded — these
//     do not need a producer in the map) or has a producer registered in
//     producers — this lets the client gracefully skip unregistered
//     spec entries instead of erroring at the gate that follows;
//  3. the first non-empty entry overall (preserves the historical error
//     path: the gate at the call site reports "none of producers" with
//     the unregistered mime, so the diagnostic is unchanged when nothing
//     in consumes is registered);
//  4. def, if consumes is empty or all empty strings.
//
// Step 2 closes part of issues #32 and #386: an operation declaring
// `consumes: [application/x-vendor, application/json]` with no vendor
// producer registered now silently uses JSON instead of erroring.
func pickConsumesMediaType(consumes []string, producers map[string]runtime.Producer, def string) string {
	for _, mt := range consumes {
		if strings.EqualFold(mt, runtime.MultipartFormMime) {
			return mt
		}
	}
	var firstNonEmpty string
	for _, mt := range consumes {
		if mt == "" {
			continue
		}
		if firstNonEmpty == "" {
			firstNonEmpty = mt
		}
		if isStructuralMime(mt) {
			return mt
		}
		if _, ok := producers[mt]; ok {
			return mt
		}
	}
	if firstNonEmpty != "" {
		return firstNonEmpty
	}
	return def
}

// isStructuralMime reports whether mt is a media type whose body shape
// is owned by the runtime (multipart envelope, urlencoded form). These
// do not require an entry in the producers map.
func isStructuralMime(mt string) bool {
	return strings.EqualFold(mt, runtime.MultipartFormMime) ||
		strings.EqualFold(mt, runtime.URLencodedFormMime)
}
