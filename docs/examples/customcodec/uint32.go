// SPDX-License-Identifier: Apache-2.0

// Package customcodec illustrates how to implement a runtime.Consumer for a
// custom wire format. Uint32Consumer decodes a single big-endian 32-bit
// unsigned integer from the request body into a *uint32 target.
package customcodec

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
)

// Uint32Consumer returns a runtime.Consumer that reads a single big-endian
// uint32 from r and stores it at v (which must be a *uint32).
func Uint32Consumer() runtime.Consumer {
	return runtime.ConsumerFunc(func(r io.Reader, v any) error {
		p, ok := v.(*uint32)
		if !ok {
			return fmt.Errorf("uint32 consumer: target %T is not *uint32", v)
		}
		return binary.Read(r, binary.BigEndian, p)
	})
}
