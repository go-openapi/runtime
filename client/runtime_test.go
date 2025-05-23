// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token       = "the-super-secret-token"
	bearerToken = "Bearer " + token
	charsetUTF8 = ";charset=utf-8"
)

// task This describes a task. Tasks require a content property to be set.
type task struct {
	// Completed
	Completed bool `json:"completed" xml:"completed"`

	// Content Task content can contain [GFM](https://help.github.com/articles/github-flavored-markdown/).
	Content string `json:"content" xml:"content"`

	// ID This id property is autogenerated when a task is created.
	ID int64 `json:"id" xml:"id"`
}

type testCtxKey uint8

const rtKey testCtxKey = 1

func TestRuntime_Concurrent(t *testing.T) {
	// test that it can make a simple request
	// and get the response for it.
	// defaults all the way down
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		assert.NoError(t, jsongen.Encode(result))
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	resCC := make(chan interface{})
	errCC := make(chan error)
	var res interface{}

	for j := 0; j < 6; j++ {
		go func() {
			resC := make(chan interface{})
			errC := make(chan error)

			go func() {
				var resp interface{}
				var errp error
				for i := 0; i < 3; i++ {
					resp, errp = rt.Submit(&runtime.ClientOperation{
						ID:          "getTasks",
						Method:      http.MethodGet,
						PathPattern: "/",
						Params:      rwrtr,
						Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
							if response.Code() == http.StatusOK {
								var res []task
								if e := consumer.Consume(response.Body(), &res); e != nil {
									return nil, e
								}
								return res, nil
							}
							return nil, errors.New("generic error")
						}),
					})
					<-time.After(100 * time.Millisecond)
				}
				resC <- resp
				errC <- errp
			}()
			resCC <- <-resC
			errCC <- <-errC
		}()
	}

	c := 6
	for c > 0 {
		res = <-resCC
		err = <-errCC
		c--
	}

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_Canary(t *testing.T) {
	// test that it can make a simple request
	// and get the response for it.
	// defaults all the way down
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		assert.NoError(t, jsongen.Encode(result))
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)
	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []task
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

type tasks struct {
	Tasks []task `xml:"task"`
}

func TestRuntime_XMLCanary(t *testing.T) {
	// test that it can make a simple XML request
	// and get the response for it.
	result := tasks{
		Tasks: []task{
			{false, "task 1 content", 1},
			{false, "task 2 content", 2},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, runtime.XMLMime)
		rw.WriteHeader(http.StatusOK)
		xmlgen := xml.NewEncoder(rw)
		assert.NoError(t, xmlgen.Encode(result))
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res tasks
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})

	require.NoError(t, err)
	assert.IsType(t, tasks{}, res)
	actual := res.(tasks)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_TextCanary(t *testing.T) {
	// test that it can make a simple text request
	// and get the response for it.
	result := "1: task 1 content; 2: task 2 content"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, runtime.TextMime)
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(result))
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res string
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})

	require.NoError(t, err)
	assert.IsType(t, "", res)
	actual := res.(string)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_CSVCanary(t *testing.T) {
	// test that it can make a simple csv request
	// and get the response for it.
	result := `task,content,result
1,task1,ok
2,task2,fail
`
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, runtime.CSVMime)
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(result))
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res bytes.Buffer
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})

	require.NoError(t, err)
	assert.IsType(t, bytes.Buffer{}, res)
	actual := res.(bytes.Buffer)
	assert.EqualValues(t, result, actual.String())
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRuntime_CustomTransport(t *testing.T) {
	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}

	rt := New("localhost:3245", "/", []string{"ws", "wss", schemeHTTPS})
	rt.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Scheme != schemeHTTPS {
			return nil, errors.New("this was not a https request")
		}
		assert.Equal(t, "localhost:3245", req.Host)
		assert.Equal(t, "localhost:3245", req.URL.Host)

		var resp http.Response
		resp.StatusCode = http.StatusOK
		resp.Header = make(http.Header)
		resp.Header.Set("Content-Type", "application/json")
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		require.NoError(t, enc.Encode(result))
		resp.Body = io.NopCloser(buf)
		return &resp, nil
	})

	res, err := rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Schemes:     []string{"ws", "wss", schemeHTTPS},
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []task
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_CustomCookieJar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		authenticated := false
		for _, cookie := range req.Cookies() {
			if cookie.Name == "sessionid" && cookie.Value == "abc" {
				authenticated = true
			}
		}
		if !authenticated {
			username, password, ok := req.BasicAuth()
			if ok && username == "username" && password == "password" {
				authenticated = true
				http.SetCookie(rw, &http.Cookie{Name: "sessionid", Value: "abc"})
			}
		}
		if authenticated {
			rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime)
			rw.WriteHeader(http.StatusOK)
			jsongen := json.NewEncoder(rw)
			assert.NoError(t, jsongen.Encode([]task{}))
		} else {
			rw.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	rt.Jar, err = cookiejar.New(nil)
	require.NoError(t, err)

	submit := func(authInfo runtime.ClientAuthInfoWriter) {
		_, err := rt.Submit(&runtime.ClientOperation{
			ID:          "getTasks",
			Method:      http.MethodGet,
			PathPattern: "/",
			Params:      rwrtr,
			AuthInfo:    authInfo,
			Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, _ runtime.Consumer) (interface{}, error) {
				if response.Code() == http.StatusOK {
					return map[string]interface{}{}, nil
				}
				return nil, errors.New("generic error")
			}),
		})

		require.NoError(t, err)
	}

	submit(BasicAuth("username", "password"))
	submit(nil)
}

func TestRuntime_AuthCanary(t *testing.T) {

	// test that it can make a simple request
	// and get the response for it.
	// defaults all the way down
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(runtime.HeaderAuthorization) != bearerToken {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		assert.NoError(t, jsongen.Encode(result))
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:     "getTasks",
		Params: rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []task
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
		AuthInfo: BearerToken(token),
	})

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_PickConsumer(t *testing.T) {
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/octet-stream" {
			rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime+charsetUTF8)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime+charsetUTF8)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		_ = jsongen.Encode(result)
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetBodyParam(bytes.NewBufferString("hello"))
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:                 "getTasks",
		Method:             "POST",
		PathPattern:        "/",
		Schemes:            []string{schemeHTTP},
		ConsumesMediaTypes: []string{"application/octet-stream"},
		Params:             rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []task
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
		AuthInfo: BearerToken(token),
	})

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_ContentTypeCanary(t *testing.T) {
	// test that it can make a simple request
	// and get the response for it.
	// defaults all the way down
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(runtime.HeaderAuthorization) != bearerToken {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime+charsetUTF8)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		_ = jsongen.Encode(result)
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Schemes:     []string{schemeHTTP},
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []task
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
		AuthInfo: BearerToken(token),
	})

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_ChunkedResponse(t *testing.T) {
	// test that it can make a simple request
	// and get the response for it.
	// defaults all the way down
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(runtime.HeaderAuthorization) != bearerToken {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.Header().Add(runtime.HeaderTransferEncoding, "chunked")
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime+charsetUTF8)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		_ = jsongen.Encode(result)
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	// specDoc, err := spec.Load("../../fixtures/codegen/todolist.simple.yml")
	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/",
		Schemes:     []string{schemeHTTP},
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []task
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
		AuthInfo: BearerToken(token),
	})

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_DebugValue(t *testing.T) {
	t.Run("empty DEBUG means Debug is False", func(t *testing.T) {
		t.Setenv("DEBUG", "")

		runtime := New("", "/", []string{schemeHTTPS})
		assert.False(t, runtime.Debug)
	})

	t.Run("non-Empty DEBUG means Debug is True", func(t *testing.T) {
		t.Run("with numerical value", func(t *testing.T) {
			t.Setenv("DEBUG", "1")

			runtime := New("", "/", []string{schemeHTTPS})
			assert.True(t, runtime.Debug)
		})

		t.Run("with boolean value true", func(t *testing.T) {
			t.Setenv("DEBUG", "true")

			runtime := New("", "/", []string{schemeHTTPS})
			assert.True(t, runtime.Debug)
		})

		t.Run("with boolean value false", func(t *testing.T) {
			t.Setenv("DEBUG", "false")

			runtime := New("", "/", []string{schemeHTTPS})
			assert.False(t, runtime.Debug)
		})

		t.Run("with string value ", func(t *testing.T) {
			t.Setenv("DEBUG", "foo")

			runtime := New("", "/", []string{schemeHTTPS})
			assert.True(t, runtime.Debug)
		})
	})
}

func TestRuntime_OverrideScheme(t *testing.T) {
	runtime := New("", "/", []string{schemeHTTPS})
	sch := runtime.pickScheme([]string{schemeHTTP})
	assert.Equal(t, schemeHTTPS, sch)
}

func TestRuntime_OverrideClient(t *testing.T) {
	client := &http.Client{}
	runtime := NewWithClient("", "/", []string{schemeHTTPS}, client)
	var i int
	runtime.clientOnce.Do(func() { i++ })
	assert.Equal(t, client, runtime.client)
	assert.Equal(t, 0, i)
}

type overrideRoundTripper struct {
	overridden bool
}

func (o *overrideRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	o.overridden = true
	res := new(http.Response)
	res.StatusCode = http.StatusOK
	res.Body = io.NopCloser(bytes.NewBufferString("OK"))
	return res, nil
}

func TestRuntime_OverrideClientOperation(t *testing.T) {
	client := &http.Client{}
	rt := NewWithClient("", "/", []string{schemeHTTPS}, client)
	var i int
	rt.clientOnce.Do(func() { i++ })
	assert.Equal(t, client, rt.client)
	assert.Equal(t, 0, i)

	client2 := new(http.Client)
	var transport = &overrideRoundTripper{}
	client2.Transport = transport
	require.NotEqual(t, client, client2)

	_, err := rt.Submit(&runtime.ClientOperation{
		Client: client2,
		Params: runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
			return nil
		}),
		Reader: runtime.ClientResponseReaderFunc(func(_ runtime.ClientResponse, _ runtime.Consumer) (interface{}, error) {
			return map[string]interface{}{}, nil
		}),
	})
	require.NoError(t, err)
	assert.True(t, transport.overridden)
}

func TestRuntime_PreserveTrailingSlash(t *testing.T) {
	var redirected bool

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime+charsetUTF8)

		if req.URL.Path == "/api/tasks" {
			redirected = true
			return
		}
		if req.URL.Path == "/api/tasks/" {
			rw.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	rwrtr := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
		return nil
	})

	_, err = rt.Submit(&runtime.ClientOperation{
		ID:          "getTasks",
		Method:      http.MethodGet,
		PathPattern: "/api/tasks/",
		Params:      rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, _ runtime.Consumer) (interface{}, error) {
			if redirected {
				return nil, errors.New("expected Submit to preserve trailing slashes - this caused a redirect")
			}
			if response.Code() == http.StatusOK {
				return map[string]interface{}{}, nil
			}
			return nil, errors.New("generic error")
		}),
	})
	require.NoError(t, err)
}

func TestRuntime_FallbackConsumer(t *testing.T) {
	result := `W3siY29tcGxldGVkIjpmYWxzZSwiY29udGVudCI6ImRHRnpheUF4SUdOdmJuUmxiblE9IiwiaWQiOjF9XQ==`
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Add(runtime.HeaderContentType, "application/x-task")
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(result))
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetBodyParam(bytes.NewBufferString("hello"))
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)
	rt := New(hu.Host, "/", []string{schemeHTTP})

	// without the fallback consumer
	_, err = rt.Submit(&runtime.ClientOperation{
		ID:                 "getTasks",
		Method:             "POST",
		PathPattern:        "/",
		Schemes:            []string{schemeHTTP},
		ConsumesMediaTypes: []string{"application/octet-stream"},
		Params:             rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []byte
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})
	require.Error(t, err)
	assert.Equal(t, `no consumer: "application/x-task"`, err.Error())

	// add the fallback consumer
	rt.Consumers["*/*"] = rt.Consumers[runtime.DefaultMime]
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:                 "getTasks",
		Method:             "POST",
		PathPattern:        "/",
		Schemes:            []string{schemeHTTP},
		ConsumesMediaTypes: []string{"application/octet-stream"},
		Params:             rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []byte
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})

	require.NoError(t, err)
	assert.IsType(t, []byte{}, res)
	actual := res.([]byte)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_AuthHeaderParamDetected(t *testing.T) {
	// test that it can make a simple request
	// and get the response for it.
	// defaults all the way down
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(runtime.HeaderAuthorization) != bearerToken {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime)
		rw.WriteHeader(http.StatusOK)
		jsongen := json.NewEncoder(rw)
		_ = jsongen.Encode(result)
	}))
	defer server.Close()

	rwrtr := runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
		return req.SetHeaderParam(runtime.HeaderAuthorization, bearerToken)
	})

	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	rt := New(hu.Host, "/", []string{schemeHTTP})
	rt.DefaultAuthentication = BearerToken("not-the-super-secret-token")
	res, err := rt.Submit(&runtime.ClientOperation{
		ID:     "getTasks",
		Params: rwrtr,
		Reader: runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
			if response.Code() == http.StatusOK {
				var res []task
				if e := consumer.Consume(response.Body(), &res); e != nil {
					return nil, e
				}
				return res, nil
			}
			return nil, errors.New("generic error")
		}),
	})

	require.NoError(t, err)
	assert.IsType(t, []task{}, res)
	actual := res.([]task)
	assert.EqualValues(t, result, actual)
}

func TestRuntime_Timeout(t *testing.T) { //nolint:maintidx // linter evaluates the total lines of code, which is misleading
	const (
		operationID = "getTasks"

		// these values should be sufficient for most CI engines
		clientTimeout   time.Duration = 25 * time.Millisecond
		serverDelay     time.Duration = 100 * time.Millisecond
		clientNoTimeout time.Duration = 250 * time.Millisecond
		ctxError                      = "context deadline exceeded"
	)
	result := []task{
		{false, "task 1 content", 1},
		{false, "task 2 content", 2},
	}

	signedContext := func(value string) context.Context {
		return context.WithValue(context.Background(), rtKey, value)
	}

	requestWriter := func(timeout time.Duration) runtime.ClientRequestWriter {
		// this writer sets the timeout parameter of the ClientRequest
		return runtime.ClientRequestWriterFunc(func(req runtime.ClientRequest, _ strfmt.Registry) error {
			return req.SetTimeout(timeout)
		})
	}

	requestReader := runtime.ClientResponseReaderFunc(func(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
		if response.Code() != http.StatusOK {
			return nil, errors.New("generic error")
		}

		var res []task
		if e := consumer.Consume(response.Body(), &res); e != nil {
			return nil, e
		}
		return res, nil
	})

	t.Run("with timeout specified as a request parameter, no operation context", func(t *testing.T) {
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)

		rt := New(host, "/", []string{schemeHTTP})
		rt.Context = signedContext("test")
		rt.Transport = testContextTransport(t, true, true, "test")

		t.Run("should not time out", func(t *testing.T) {
			res, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestWriter(clientNoTimeout),
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, res)
		})

		t.Run("should time out", func(t *testing.T) {
			_, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestWriter(clientTimeout),
				Reader:  requestReader,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ctxError)
		})
	})

	t.Run("with timeout specified as a request parameter, no context at all", func(t *testing.T) {
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)

		rt := New(host, "/", []string{schemeHTTP})
		rt.Context = nil
		rt.Transport = testContextTransport(t, true, false, "")

		t.Run("should not time out", func(t *testing.T) {
			res, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestWriter(clientNoTimeout),
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, res)
		})

		t.Run("should time out", func(t *testing.T) {
			_, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestWriter(clientTimeout),
				Reader:  requestReader,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ctxError)
		})
	})

	t.Run("with inherited operation context, timeout specified as operation context, request timeout set to 0", func(t *testing.T) {
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)

		rt := New(host, "/", []string{schemeHTTP})
		rt.Context = signedContext("test")
		rt.Transport = testContextTransport(t, true, true, "test")

		t.Run("should not time out", func(t *testing.T) {
			operationCtx, cancel := context.WithTimeout(rt.Context, clientNoTimeout)
			defer cancel()

			res, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: operationCtx,
				Params:  requestWriter(0),
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, res)
		})

		t.Run("should time out", func(t *testing.T) {
			operationCtx, cancel := context.WithTimeout(rt.Context, clientTimeout)
			defer cancel()

			_, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: operationCtx,
				Params:  requestWriter(0),
				Reader:  requestReader,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ctxError)
		})
	})

	t.Run("with a fresh operation context, timeout specified as operation context, request timeout set to 0", func(t *testing.T) {
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)
		rt := New(host, "/", []string{schemeHTTP})
		rt.Context = nil
		rt.Transport = testContextTransport(t, true, false, "")

		t.Run("should not time out", func(t *testing.T) {
			operationCtx, cancel := context.WithTimeout(context.Background(), clientNoTimeout)
			defer cancel()

			res, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: operationCtx,
				Params:  requestWriter(0),
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, res)
		})

		t.Run("should time out", func(t *testing.T) {
			operationCtx, cancel := context.WithTimeout(context.Background(), clientTimeout)
			defer cancel()

			_, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: operationCtx,
				Params:  requestWriter(0),
				Reader:  requestReader,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ctxError)
		})
	})

	t.Run("with an hypothetical timeout specified as runtime context, no operation context", func(t *testing.T) {
		// in real life, the runtime context may be cancellable for other reasons than timeout
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)

		t.Run("should not time out", func(t *testing.T) {
			rt := New(host, "/", []string{schemeHTTP})
			ctx, cancel := context.WithTimeout(signedContext("test"), clientNoTimeout)
			defer cancel()

			rt.Context = ctx
			rt.Transport = testContextTransport(t, true, true, "test")

			res, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestWriter(0),
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, res)
		})

		t.Run("should time out", func(t *testing.T) {
			rt := New(host, "/", []string{schemeHTTP})
			ctx, cancel := context.WithTimeout(signedContext("test"), clientTimeout)
			defer cancel()

			rt.Context = ctx
			rt.Transport = testContextTransport(t, true, true, "test")

			_, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestWriter(0),
				Reader:  requestReader,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ctxError)
		})
	})

	t.Run("with multiple timeouts set, shortest wins", func(t *testing.T) {
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)

		rt := New(host, "/", []string{schemeHTTP})
		runtimeCtx, cancelRuntime := context.WithTimeout(signedContext("test"), clientNoTimeout)
		rt.Context = runtimeCtx
		defer cancelRuntime()
		rt.Transport = testContextTransport(t, true, true, "test")

		t.Run("should not time out", func(t *testing.T) {
			operationCtx, cancelOperation := context.WithTimeout(
				signedContext("test"),
				serverDelay+(clientNoTimeout-serverDelay)/2,
			)
			defer cancelOperation()

			res, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: operationCtx,
				Params:  requestWriter(serverDelay + (clientNoTimeout-serverDelay)/3),
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, res)
		})

		t.Run("should time out on operation context deadline", func(t *testing.T) {
			// NOTE: we'll be able to catch more precisely which context was canceled
			// in go1.21 and context.WithTimeoutCause.
			operationCtx, cancelOperation := context.WithTimeout(
				signedContext("test"),
				serverDelay-(clientNoTimeout-serverDelay)/4, // this one times out
			)
			defer cancelOperation()

			_, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: operationCtx,
				Params: requestWriter(
					serverDelay + (clientNoTimeout-serverDelay)/4,
				),
				Reader: requestReader,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ctxError)
		})

		t.Run("should time out on operation timeout param", func(t *testing.T) {
			operationCtx, cancelOperation := context.WithTimeout(
				signedContext("test"),
				serverDelay+(clientNoTimeout-serverDelay)/2,
			)
			defer cancelOperation()

			_, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: operationCtx,
				Params: requestWriter(
					serverDelay - (clientNoTimeout-serverDelay)/4, // this one times out
				),
				Reader: requestReader,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ctxError)
		})
	})

	t.Run("with no context, explicit infinite wait", func(t *testing.T) {
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)

		rt := New(host, "/", []string{schemeHTTP})
		rt.Context = signedContext("test")
		rt.Transport = testContextTransport(t, false, true, "test") // verify that no deadline is passed to the emitted context

		t.Run("should not time out", func(t *testing.T) {
			resp, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestWriter(0),
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, resp)
		})
	})
	t.Run("with no context, request uses the default timeout", func(t *testing.T) {
		requestEmptyWriter := runtime.ClientRequestWriterFunc(func(_ runtime.ClientRequest, _ strfmt.Registry) error {
			return nil
		})
		host, cleaner := serverBuilder(t, serverDelay, result)
		t.Cleanup(cleaner)

		rt := New(host, "/", []string{schemeHTTP})
		rt.Context = signedContext("test")
		rt.Transport = testDefaultsInTransport(t, "test")

		t.Run("should not time out", func(t *testing.T) {
			resp, err := rt.Submit(&runtime.ClientOperation{
				ID:      operationID,
				Context: nil,
				Params:  requestEmptyWriter, // leaves defaults
				Reader:  requestReader,
			})
			require.NoError(t, err)
			assertResult(result)(t, resp)
		})
	})
}

func isContextSigned(ctx context.Context, value string) bool {
	v, ok := ctx.Value(rtKey).(string)

	return ok && v == value
}

func testContextTransport(t *testing.T, hasTimeout, expectSigned bool, value string) http.RoundTripper {

	// inject a round tripper to check the context in the request about to be emitted
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		ctx := req.Context()

		t.Run("request context should propagate value", func(t *testing.T) {
			assert.Equal(t, expectSigned, isContextSigned(ctx, value), "expected the request context to inherit values")
		})

		t.Run(fmt.Sprintf("request context should have a deadline %t", hasTimeout), func(t *testing.T) {
			_, hasDeadline := ctx.Deadline()
			assert.Equalf(t, hasTimeout, hasDeadline, "expected request context to have a deadline")
		})

		return http.DefaultTransport.RoundTrip(req)
	})
}

func testDefaultsInTransport(t *testing.T, value string) http.RoundTripper {
	// inject a round tripper to check the context in the request about to be emitted
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		ctx := req.Context()

		t.Run("request context should propagate value", func(t *testing.T) {
			assert.True(t, isContextSigned(ctx, value), "expected the request context to inherit values")
		})

		t.Run("request context should have a default deadline", func(t *testing.T) {
			deadline, hasDeadline := ctx.Deadline()
			assert.True(t, hasDeadline, "expected request context to have a deadline")

			remainingDuration := time.Until(deadline).Seconds()
			assert.InDeltaf(t, DefaultTimeout.Seconds(), remainingDuration, 1.0, "expected timeout to be set to DefaultTimeout")
		})

		return http.DefaultTransport.RoundTrip(req)
	})
}

func assertResult(result []task) func(testing.TB, interface{}) {
	return func(t testing.TB, res interface{}) {
		assert.IsType(t, []task{}, res)
		actual, ok := res.([]task)
		require.True(t, ok)
		assert.EqualValues(t, result, actual)
	}
}

func serverBuilder(t testing.TB, delay time.Duration, result []task) (string, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			http.Error(rw, ctx.Err().Error(), http.StatusInternalServerError)

			return
		case <-timer.C:
			rw.Header().Add(runtime.HeaderContentType, runtime.JSONMime)
			rw.WriteHeader(http.StatusOK)
			jsongen := json.NewEncoder(rw)
			_ = jsongen.Encode(result)

			return
		}
	}))
	hu, err := url.Parse(server.URL)
	require.NoError(t, err)

	return hu.Host, server.Close
}
