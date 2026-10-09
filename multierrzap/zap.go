// Copyright (c) 2024 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package multierrzap provides zapcore.ArrayMarshaler support and field
// constructors for multierr error inspection structures.
package multierrzap

import (
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.ArrayMarshaler = (*ErrorArray)(nil)

// ErrorArray wraps a slice of errors extracted from a multi-error inspection,
// implementing zapcore.ArrayMarshaler to serialize each underlying error.
type ErrorArray []error

// Errors returns a zapcore.ArrayMarshaler for the underlying list of errors
// extracted from the supplied error. If err is nil, an empty ErrorArray is returned.
func Errors(err error) ErrorArray {
	return ErrorArray(multierr.Errors(err))
}

// Inspect returns an ErrorArray containing all underlying errors extracted from
// the supplied error, implementing zapcore.ArrayMarshaler.
func Inspect(err error) ErrorArray {
	return Errors(err)
}

// MarshalLogArray implements zapcore.ArrayMarshaler for ErrorArray.
func (ea ErrorArray) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	for _, err := range ea {
		if err == nil {
			continue
		}
		if el, ok := err.(zapcore.ArrayMarshaler); ok {
			if err := arr.AppendArray(el); err != nil {
				return err
			}
		} else if el, ok := err.(zapcore.ObjectMarshaler); ok {
			if err := arr.AppendObject(el); err != nil {
				return err
			}
		} else {
			arr.AppendString(err.Error())
		}
	}
	return nil
}

// Field constructs a zap.Field encoding the inspected errors under the given
// key using zapcore.ArrayMarshaler.
func Field(key string, err error) zap.Field {
	return zap.Array(key, Errors(err))
}
