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

package runtime

import (
	"github.com/gorilla/schema"
	"io"
	"bytes"
	"net/url"
)

// UrlformConsumer creates a new x-www-form-urlencoded consumer
func UrlformConsumer() Consumer {
	return ConsumerFunc(func(reader io.Reader, data interface{}) error {
		var err error
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(reader)
		if err != nil {
			return err
		}

		parsed_data, err := url.ParseQuery(buf.String())
		if err != nil {
			return err
		}

		decoder := schema.NewDecoder()
		decoder.SetAliasTag("json")
		// data is already a pointer to a struct
		err = decoder.Decode(data, parsed_data)
		if err != nil {
			return err
		}

		return nil
	})
}

// UrlformProducer creates a new x-www-form-urlencoded producer
func UrlformProducer() Producer {
	return ProducerFunc(func(writer io.Writer, data interface{}) error {
		var err error
		dst := make(url.Values)
		encoder := schema.NewEncoder()
		encoder.SetAliasTag("json")
		err = encoder.Encode(data, dst)
		if err != nil {
			return err
		}

		_, err = writer.Write([]byte(dst.Encode()))
		if err != nil {
			return err
		}

		return nil
	})
}
